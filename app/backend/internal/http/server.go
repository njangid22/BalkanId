package http

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"vault/graph"
	"vault/internal/auth"
	"vault/internal/config"
	"vault/internal/db"
	"vault/internal/files"
)

type Server struct {
	cfg          config.Config
	router       chi.Router
	db           *db.Pool
	fileSvc      *files.Service
	oauth        *auth.GoogleOAuth
	jwt          *auth.JWTManager
	stateCookie  string
	secureCookie bool
	limiter      *rateLimiter
}

func NewServer(cfg config.Config, pool *db.Pool, fileSvc *files.Service, oauth *auth.GoogleOAuth, jwtMgr *auth.JWTManager) *Server {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	origin := strings.TrimSuffix(cfg.FrontendURL, "/")
	if origin == "" {
		origin = "http://localhost:3000"
	}
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{origin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	server := &Server{
		cfg:          cfg,
		router:       router,
		db:           pool,
		fileSvc:      fileSvc,
		oauth:        oauth,
		jwt:          jwtMgr,
		stateCookie:  "vault_oauth_state",
		secureCookie: strings.HasPrefix(strings.ToLower(cfg.FrontendURL), "https://"),
		limiter:      newRateLimiter(cfg.RateLimitRPS),
	}

	router.Use(server.rateLimitMiddleware())
	server.registerRoutes()
	return server
}

func (s *Server) registerRoutes() {
	s.router.Get("/healthz", s.handleHealth)
	s.router.Get("/auth/google/start", s.handleGoogleStart)
	s.router.Get("/auth/google/callback", s.handleGoogleCallback)

	s.router.Route("/files", func(r chi.Router) {
		r.Get("/{fileID}/download", s.handleFileDownload)
		r.Get("/{fileID}/share", s.handleShareInfo)
	})
	s.router.Get("/shares/{token}/download", s.handleShareDownload)

	// Public download by file ID: resolves associated PUBLIC share and streams content
	s.router.Get("/public/files/{fileID}/download", s.handlePublicFileDownload)

	gqlServer := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: graph.NewResolver(s.db, s.fileSvc)}))
	gqlServer.AddTransport(transport.MultipartForm{
		MaxUploadSize: s.cfg.MaxUploadBytes,
		MaxMemory:     s.cfg.MaxUploadBytes,
	})

	s.router.Handle("/graphql", s.withSession(gqlServer))
	s.router.Get("/playground", func(w http.ResponseWriter, r *http.Request) {
		playground.Handler("GraphQL", "/graphql").ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := "ok"
	if s.db != nil {
		if err := s.db.Ping(ctx); err != nil {
			status = "degraded"
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

func (s *Server) handleGoogleStart(w http.ResponseWriter, r *http.Request) {
	state, err := s.newStateToken()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     s.stateCookie,
		Value:    state,
		Path:     "/auth/google",
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(5 * time.Minute),
	})

	authURL := s.oauth.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Errorf("parse callback: %w", err))
		return
	}

	state := r.FormValue("state")
	code := r.FormValue("code")

	if !s.validateState(r, state) {
		s.writeError(w, http.StatusBadRequest, errors.New("invalid oauth state"))
		return
	}

	user, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, err)
		return
	}

	dbUser, err := s.db.UpsertUser(ctx, user.Email, user.Name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}

	token, claims, err := s.jwt.Sign(time.Now(), dbUser.ID.String(), dbUser.Email, user.Name, dbUser.Role)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Cross-site (Vercel -> Railway) requires SameSite=None; Secure and works best with Partitioned (CHIPS)
	s.setSessionCookie(w, s.cfg.SessionCookieName, token, claims.ExpiresAt.Time)

	s.clearStateCookie(w)

	// Include JWT in fragment as a fallback for browsers blocking third-party cookies.
	// The cookie is still set server-side; fragment allows frontend to store token and use Authorization header.
	redirect := strings.TrimSuffix(s.cfg.FrontendURL, "/") + "/files#token=" + url.QueryEscape(token)
	http.Redirect(w, r, redirect, http.StatusFound)
}

func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	session, err := s.sessionFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, err)
		return
	}
	if session == nil {
		s.writeError(w, http.StatusUnauthorized, errors.New("unauthenticated"))
		return
	}

	ownerID, err := uuid.Parse(session.UserID)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, fmt.Errorf("invalid session user"))
		return
	}

	fileIDParam := chi.URLParam(r, "fileID")
	fileID, err := uuid.Parse(fileIDParam)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid file id"))
		return
	}

	downloaded, err := s.fileSvc.DownloadOwnedFile(r.Context(), fileID, ownerID)
	if err != nil {
		if errors.Is(err, files.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, errors.New("file not found"))
			return
		}
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.writeFileResponse(w, downloaded)
}

func (s *Server) handleShareDownload(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		s.writeError(w, http.StatusBadRequest, errors.New("missing share token"))
		return
	}

	downloaded, err := s.fileSvc.DownloadSharedFile(r.Context(), token)
	if err != nil {
		if errors.Is(err, files.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, errors.New("share not found"))
			return
		}
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.writeFileResponse(w, downloaded)
}

// handlePublicFileDownload allows downloading a file by ID if it has a PUBLIC share.
func (s *Server) handlePublicFileDownload(w http.ResponseWriter, r *http.Request) {
	fileIDParam := chi.URLParam(r, "fileID")
	fileID, err := uuid.Parse(fileIDParam)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid file id"))
		return
	}

	share, err := s.db.GetShareByFileID(r.Context(), fileID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	if share == nil || strings.ToUpper(share.Visibility) != "PUBLIC" || share.Token == nil || *share.Token == "" {
		s.writeError(w, http.StatusNotFound, errors.New("public share not found"))
		return
	}

	downloaded, err := s.fileSvc.DownloadSharedFile(r.Context(), *share.Token)
	if err != nil {
		if errors.Is(err, files.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, errors.New("file not found"))
			return
		}
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.writeFileResponse(w, downloaded)
}

// handleShareInfo returns share details (visibility, token, expiresAt) for an owned file.
func (s *Server) handleShareInfo(w http.ResponseWriter, r *http.Request) {
	session, err := s.sessionFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, err)
		return
	}
	if session == nil {
		s.writeError(w, http.StatusUnauthorized, errors.New("unauthenticated"))
		return
	}

	ownerID, err := uuid.Parse(session.UserID)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, fmt.Errorf("invalid session user"))
		return
	}

	fileIDParam := chi.URLParam(r, "fileID")
	fileID, err := uuid.Parse(fileIDParam)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid file id"))
		return
	}

	fileWithBlob, err := s.db.GetFileWithBlob(r.Context(), fileID, ownerID)
	if err != nil || fileWithBlob == nil {
		s.writeError(w, http.StatusNotFound, errors.New("file not found"))
		return
	}

	share, err := s.db.GetShareByFileID(r.Context(), fileID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	if share == nil {
		s.writeError(w, http.StatusNotFound, errors.New("share not found"))
		return
	}

	resp := map[string]any{
		"share": map[string]any{
			"id":         share.ID.String(),
			"visibility": share.Visibility,
			"token":      share.Token,
			"expiresAt":  share.ExpiresAt,
		},
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) writeFileResponse(w http.ResponseWriter, payload *files.DownloadedFile) {
	if payload == nil {
		s.writeError(w, http.StatusInternalServerError, errors.New("missing file payload"))
		return
	}

	contentType := payload.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	filename := payload.File.FilenameOriginal
	if filename == "" {
		filename = payload.File.ID.String()
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(payload.Data)))
	w.Header().Set("Content-Disposition", buildContentDisposition(filename))
	w.Header().Set("Cache-Control", "no-store")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload.Data)
}

func buildContentDisposition(filename string) string {
	safeName := sanitizeFilename(filename)
	base := mime.FormatMediaType("attachment", map[string]string{"filename": safeName})
	escaped := url.PathEscape(filename)
	if escaped == "" {
		escaped = url.PathEscape(safeName)
	}
	return fmt.Sprintf("%s; filename*=UTF-8''%s", base, escaped)
}

func sanitizeFilename(name string) string {
	trimmed := strings.TrimSpace(name)
	sanitized := strings.Map(func(r rune) rune {
		if r == '\\' || r == '"' || r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, trimmed)
	if sanitized == "" {
		return "download"
	}
	return sanitized
}

func (s *Server) rateLimitMiddleware() func(http.Handler) http.Handler {
	if s.limiter == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			key := ""
			if session, err := s.sessionFromRequest(r); err == nil && session != nil && session.UserID != "" {
				key = "user:" + session.UserID
			} else {
				key = "ip:" + clientIPAddress(r.RemoteAddr)
			}

			if !s.limiter.Allow(key, time.Now()) {
				s.writeError(w, http.StatusTooManyRequests, errors.New("rate limit exceeded"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) withSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessionFromRequest(r)
		if err != nil {
			s.writeError(w, http.StatusUnauthorized, err)
			return
		}
		if session != nil {
			ctx := auth.WithSession(r.Context(), session)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) sessionFromRequest(r *http.Request) (*auth.Session, error) {
	// Prefer cookie if present
	if cookie, err := r.Cookie(s.cfg.SessionCookieName); err == nil && cookie != nil && cookie.Value != "" {
		if claims, err := s.jwt.Parse(cookie.Value); err == nil {
			return &auth.Session{UserID: claims.UserID, Email: claims.Email, Name: claims.Name, Role: claims.Role}, nil
		}
	}

	// Fallback: Authorization: Bearer <token>
	authz := r.Header.Get("Authorization")
	if strings.HasPrefix(authz, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
		if token != "" {
			if claims, err := s.jwt.Parse(token); err == nil {
				return &auth.Session{UserID: claims.UserID, Email: claims.Email, Name: claims.Name, Role: claims.Role}, nil
			} else {
				return nil, fmt.Errorf("parse bearer token: %w", err)
			}
		}
	}

	// No credentials
	return nil, nil
}

func (s *Server) validateState(r *http.Request, state string) bool {
	cookie, err := r.Cookie(s.stateCookie)
	if err != nil {
		return false
	}
	return cookie.Value != "" && cookie.Value == state
}

func (s *Server) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.stateCookie,
		Value:    "",
		Path:     "/auth/google",
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (s *Server) newStateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *Server) writeError(w http.ResponseWriter, code int, err error) {
	if err == nil {
		err = errors.New("unknown error")
	}
	s.writeJSON(w, code, map[string]string{"error": err.Error()})
}

func (s *Server) writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.cfg.Port)
	return http.ListenAndServe(addr, s.router)
}

// setSessionCookie writes the session cookie with attributes suitable for cross-site usage.
// If the frontend is HTTPS (s.secureCookie), we set SameSite=None; Secure and add the Partitioned attribute (CHIPS)
// to improve compatibility when third-party cookies are restricted by the browser.
func (s *Server) setSessionCookie(w http.ResponseWriter, name, value string, expires time.Time) {
	// Default to Lax for same-site local development
	sameSite := http.SameSiteLaxMode
	if s.secureCookie {
		sameSite = http.SameSiteNoneMode
	}

	// Attempt to use net/http Cookie first
	base := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: sameSite,
		Expires:  expires,
	}

	// Write base cookie
	http.SetCookie(w, base)

	// Add Partitioned attribute for CHIPS, if serving over HTTPS.
	// Older Go versions don't have Cookie.Partitioned; appending a second header with the attribute works.
	if s.secureCookie {
		// Rebuild cookie string to append "; Partitioned" once.
		// Format time as per RFC1123 with GMT timezone.
		expiresStr := expires.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
		// Note: Do not set Domain so the cookie is host-only for the backend host.
		cookieStr := fmt.Sprintf("%s=%s; Path=/; Expires=%s; HttpOnly; Secure; SameSite=None; Partitioned", name, value, expiresStr)
		w.Header().Add("Set-Cookie", cookieStr)
	}
}
