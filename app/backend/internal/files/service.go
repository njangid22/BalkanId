package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"vault/internal/db"
	"vault/internal/storage"
)

// UploadInput represents an incoming file stream to be stored.
type UploadInput struct {
	Filename     string
	DeclaredMIME string
	Reader       io.Reader
	Size         int64
}

type Service struct {
	repo           *db.Pool
	storage        *storage.SupabaseClient
	maxUploadBytes int64
}

var ErrNotFound = errors.New("file not found")

type DownloadedFile struct {
	File        db.FileRecord
	Blob        db.FileBlob
	Data        []byte
	ContentType string
}

func NewService(repo *db.Pool, storage *storage.SupabaseClient, maxUploadBytes int64) *Service {
	return &Service{repo: repo, storage: storage, maxUploadBytes: maxUploadBytes}
}

// UploadResult contains metadata for the created file records.
type UploadResult struct {
	File  db.FileRecord
	Blob  db.FileBlob
	IsNew bool
}

func (s *Service) Upload(ctx context.Context, owner db.User, inputs []UploadInput) ([]UploadResult, error) {
	results := make([]UploadResult, 0, len(inputs))

	originalUsage, _, err := s.repo.StorageUsage(ctx, owner.ID)
	if err != nil {
		return nil, err
	}

	for _, input := range inputs {
		data, hash, detectedMIME, err := readAndHash(input.Reader, input.DeclaredMIME)
		if err != nil {
			return nil, err
		}
		size := int64(len(data))

		if s.maxUploadBytes > 0 && size > s.maxUploadBytes {
			return nil, fmt.Errorf("file %s exceeds max upload size of %d bytes", input.Filename, s.maxUploadBytes)
		}

		if owner.QuotaBytes > 0 && originalUsage+size > owner.QuotaBytes {
			return nil, fmt.Errorf("storage quota exceeded")
		}

		blob, err := s.repo.GetBlobByHash(ctx, hash)
		if err != nil {
			return nil, err
		}

		storageKey := buildStorageKey(hash)
		isNew := false
		if blob == nil {
			if err := s.storage.Upload(ctx, storageKey, data, detectedMIME); err != nil {
				return nil, err
			}
			blob, err = s.repo.InsertBlob(ctx, hash, size, detectedMIME, storageKey)
			if err != nil {
				return nil, err
			}
			isNew = true
		} else {
			if err := s.repo.IncrementBlobRef(ctx, blob.ID); err != nil {
				return nil, err
			}
			blob.RefCount++
		}

		record := &db.FileRecord{
			OwnerID:            owner.ID,
			BlobID:             blob.ID,
			FilenameOriginal:   input.Filename,
			FilenameNormalized: strings.ToLower(input.Filename),
			SizeBytesOriginal:  size,
			Tags:               []string{},
		}
		if input.DeclaredMIME != "" {
			declared := input.DeclaredMIME
			record.MimeDeclared = &declared
		}

		if err := s.repo.InsertFile(ctx, record); err != nil {
			return nil, err
		}

		results = append(results, UploadResult{File: *record, Blob: *blob, IsNew: isNew})
		originalUsage += size
	}

	return results, nil
}

func readAndHash(r io.Reader, declaredMIME string) ([]byte, string, string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", "", err
	}

	hash := sha256.Sum256(data)
	hashHex := hex.EncodeToString(hash[:])

	detected := http.DetectContentType(sampleBytes(data))
	if declaredMIME != "" && !strings.EqualFold(declaredMIME, detected) {
		if detected == "application/octet-stream" {
			detected = declaredMIME
		}
	}

	return data, hashHex, detected, nil
}

func sampleBytes(data []byte) []byte {
	if len(data) < 512 {
		return data
	}
	return data[:512]
}

func buildStorageKey(hash string) string {
	if len(hash) < 4 {
		return fmt.Sprintf("sha256/%s", hash)
	}
	return fmt.Sprintf("sha256/%s/%s/%s", hash[:2], hash[2:4], hash)
}

func (s *Service) DownloadOwnedFile(ctx context.Context, fileID, ownerID uuid.UUID) (*DownloadedFile, error) {
	fileWithBlob, err := s.repo.GetFileWithBlob(ctx, fileID, ownerID)
	if err != nil {
		return nil, err
	}
	if fileWithBlob == nil {
		return nil, ErrNotFound
	}

	data, contentType, err := s.storage.Download(ctx, fileWithBlob.Blob.StorageKey)
	if err != nil {
		return nil, err
	}

	if err := s.repo.IncrementDownload(ctx, fileWithBlob.File.ID); err != nil {
		return nil, err
	}

	return &DownloadedFile{
		File:        fileWithBlob.File,
		Blob:        fileWithBlob.Blob,
		Data:        data,
		ContentType: resolveContentType(contentType, fileWithBlob.File, fileWithBlob.Blob),
	}, nil
}

func (s *Service) DownloadSharedFile(ctx context.Context, token string) (*DownloadedFile, error) {
	fileRec, blobRec, _, err := s.repo.GetFileByShareToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if fileRec == nil || blobRec == nil {
		return nil, ErrNotFound
	}

	data, contentType, err := s.storage.Download(ctx, blobRec.StorageKey)
	if err != nil {
		return nil, err
	}

	if err := s.repo.IncrementDownload(ctx, fileRec.ID); err != nil {
		return nil, err
	}

	return &DownloadedFile{
		File:        *fileRec,
		Blob:        *blobRec,
		Data:        data,
		ContentType: resolveContentType(contentType, *fileRec, *blobRec),
	}, nil
}

func resolveContentType(contentType string, file db.FileRecord, blob db.FileBlob) string {
	if contentType != "" {
		return contentType
	}
	if file.MimeDeclared != nil && *file.MimeDeclared != "" {
		return *file.MimeDeclared
	}
	if blob.MimeDetected != "" {
		return blob.MimeDetected
	}
	return "application/octet-stream"
}
func (s *Service) DeleteFile(ctx context.Context, fileID, ownerID uuid.UUID) (*db.FileRecord, error) {
	fileWithBlob, err := s.repo.GetFileWithBlob(ctx, fileID, ownerID)
	if err != nil || fileWithBlob == nil {
		return nil, err
	}

	if _, err := s.repo.MarkFileDeleted(ctx, fileID, ownerID); err != nil {
		return nil, err
	}

	refCount, err := s.repo.DecrementBlobRef(ctx, fileWithBlob.Blob.ID)
	if err != nil {
		return nil, err
	}

	if refCount <= 0 {
		if err := s.repo.DeleteBlob(ctx, fileWithBlob.Blob.ID); err != nil {
			return nil, err
		}
		if err := s.storage.Delete(ctx, fileWithBlob.Blob.StorageKey); err != nil {
			return nil, err
		}
	}

	_ = s.repo.DeleteShare(ctx, fileID)

	return &fileWithBlob.File, nil
}

func (s *Service) ShareFile(ctx context.Context, fileID uuid.UUID, visibility string, token *string, expires *time.Time) (*db.ShareRecord, error) {
	return s.repo.UpsertShare(ctx, fileID, visibility, token, expires)
}

func (s *Service) RevokeShare(ctx context.Context, fileID uuid.UUID) error {
	return s.repo.DeleteShare(ctx, fileID)
}

func (s *Service) StorageStats(ctx context.Context, ownerID uuid.UUID) (int64, int64, error) {
	return s.repo.StorageUsage(ctx, ownerID)
}

func (s *Service) ListFiles(ctx context.Context, ownerID uuid.UUID, filter *db.FileFilter) ([]db.FileWithBlob, int, error) {
	return s.repo.ListFiles(ctx, ownerID, filter)
}

func (s *Service) ListPublicFiles(ctx context.Context, filter *db.FileFilter) ([]db.FileWithBlob, int, error) {
	return s.repo.ListPublicFiles(ctx, filter)
}
