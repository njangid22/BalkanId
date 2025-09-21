package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"vault/internal/config"
)

const googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

// GoogleOAuth wraps the OAuth 2.0 flow for Google sign-in.
type GoogleOAuth struct {
	config *oauth2.Config
	http   *http.Client
}

// GoogleUser represents the subset of Google profile fields we rely on.
type GoogleUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// NewGoogleOAuth constructs an OAuth helper using project configuration.
func NewGoogleOAuth(cfg config.Config) (*GoogleOAuth, error) {
	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		return nil, errors.New("google oauth client not configured")
	}

	redirect := cfg.OAuthRedirectURL
	if redirect == "" {
		redirect = fmt.Sprintf("http://localhost:%s/auth/google/callback", cfg.Port)
	}

	return &GoogleOAuth{
		config: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  redirect,
			Scopes: []string{
				"openid",
				"email",
				"profile",
			},
			Endpoint: google.Endpoint,
		},
		http: http.DefaultClient,
	}, nil
}

// AuthCodeURL returns the Google authorization URL for the provided state token.
func (g *GoogleOAuth) AuthCodeURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange verifies the OAuth code and retrieves basic profile information.
func (g *GoogleOAuth) Exchange(ctx context.Context, code string) (*GoogleUser, error) {
	if strings.TrimSpace(code) == "" {
		return nil, errors.New("empty authorization code")
	}

	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build userinfo request: %w", err)
	}
	token.SetAuthHeader(req)

	resp, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("userinfo request failed: %s", resp.Status)
	}

	var user GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}

	if user.Email == "" {
		return nil, errors.New("google profile missing email")
	}

	return &user, nil
}
