package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type FileBlob struct {
	ID           uuid.UUID
	Sha256       string
	SizeBytes    int64
	MimeDetected string
	StorageKey   string
	RefCount     int
	CreatedAt    time.Time
}

type FileRecord struct {
	ID                 uuid.UUID
	OwnerID            uuid.UUID
	BlobID             uuid.UUID
	FilenameOriginal   string
	FilenameNormalized string
	MimeDeclared       *string
	SizeBytesOriginal  int64
	UploadedAt         time.Time
	IsDeleted          bool
	Tags               []string
	DownloadCount      int64
}

type FileWithBlob struct {
	File FileRecord
	Blob FileBlob
}

type ShareRecord struct {
	ID         uuid.UUID
	FileID     uuid.UUID
	Visibility string
	Token      *string
	ExpiresAt  *time.Time
}

type FileFilter struct {
	Search       *string
	MimeTypes    []string
	MinSize      *int64
	MaxSize      *int64
	Tags         []string
	UploaderName *string
	UploaderID   *uuid.UUID
	UploadedFrom *time.Time
	UploadedTo   *time.Time
}

func (p *Pool) GetBlobByHash(ctx context.Context, hash string) (*FileBlob, error) {
	const query = `
        select id, sha256, size_bytes, mime_detected, storage_key, ref_count, created_at
        from file_blobs
        where sha256 = $1
    `
	var blob FileBlob
	err := p.QueryRow(ctx, query, hash).Scan(
		&blob.ID,
		&blob.Sha256,
		&blob.SizeBytes,
		&blob.MimeDetected,
		&blob.StorageKey,
		&blob.RefCount,
		&blob.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &blob, nil
}

func (p *Pool) InsertBlob(ctx context.Context, hash string, size int64, mime, storageKey string) (*FileBlob, error) {
	const stmt = `
        insert into file_blobs (sha256, size_bytes, mime_detected, storage_key, ref_count)
        values ($1, $2, $3, $4, 1)
        returning id, created_at
    `
	var blob FileBlob
	blob.Sha256 = hash
	blob.SizeBytes = size
	blob.MimeDetected = mime
	blob.StorageKey = storageKey
	blob.RefCount = 1
	err := p.QueryRow(ctx, stmt, hash, size, mime, storageKey).Scan(&blob.ID, &blob.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &blob, nil
}

func (p *Pool) IncrementBlobRef(ctx context.Context, blobID uuid.UUID) error {
	const stmt = `update file_blobs set ref_count = ref_count + 1 where id = $1`
	_, err := p.Exec(ctx, stmt, blobID)
	return err
}

func (p *Pool) DecrementBlobRef(ctx context.Context, blobID uuid.UUID) (int, error) {
	const stmt = `
        update file_blobs
        set ref_count = ref_count - 1
        where id = $1
        returning ref_count
    `
	var refCount int
	err := p.QueryRow(ctx, stmt, blobID).Scan(&refCount)
	if err != nil {
		return 0, err
	}
	return refCount, nil
}

func (p *Pool) DeleteBlob(ctx context.Context, blobID uuid.UUID) error {
	const stmt = `delete from file_blobs where id = $1`
	_, err := p.Exec(ctx, stmt, blobID)
	return err
}

func (p *Pool) InsertFile(ctx context.Context, record *FileRecord) error {
	tagsJSON, err := json.Marshal(record.Tags)
	if err != nil {
		return err
	}

	const stmt = `
        insert into files (
            owner_id, blob_id, filename_original, filename_normalized, mime_declared,
            size_bytes_original, tags
        )
        values ($1, $2, $3, $4, $5, $6, $7)
        returning id, uploaded_at, download_count
    `
	return p.QueryRow(
		ctx,
		stmt,
		record.OwnerID,
		record.BlobID,
		record.FilenameOriginal,
		record.FilenameNormalized,
		record.MimeDeclared,
		record.SizeBytesOriginal,
		string(tagsJSON),
	).Scan(&record.ID, &record.UploadedAt, &record.DownloadCount)
}

func (p *Pool) ListFiles(ctx context.Context, ownerID uuid.UUID, filter *FileFilter) ([]FileWithBlob, int, error) {
	args := []any{ownerID}
	where := []string{"f.owner_id = $1", "f.is_deleted = false"}

	if filter != nil {
		if filter.Search != nil && *filter.Search != "" {
			args = append(args, "%"+strings.ToLower(*filter.Search)+"%")
			where = append(where, fmt.Sprintf("f.filename_normalized LIKE $%d", len(args)))
		}
		if len(filter.MimeTypes) > 0 {
			args = append(args, filter.MimeTypes)
			where = append(where, fmt.Sprintf("(coalesce(f.mime_declared, b.mime_detected) = ANY($%d))", len(args)))
		}
		if filter.MinSize != nil {
			args = append(args, *filter.MinSize)
			where = append(where, fmt.Sprintf("f.size_bytes_original >= $%d", len(args)))
		}
		if filter.MaxSize != nil {
			args = append(args, *filter.MaxSize)
			where = append(where, fmt.Sprintf("f.size_bytes_original <= $%d", len(args)))
		}
		if len(filter.Tags) > 0 {
			tagsJSON, err := json.Marshal(filter.Tags)
			if err == nil {
				args = append(args, string(tagsJSON))
				where = append(where, fmt.Sprintf("f.tags @> $%d", len(args)))
			}
		}
		if filter.UploadedFrom != nil {
			args = append(args, *filter.UploadedFrom)
			where = append(where, fmt.Sprintf("f.uploaded_at >= $%d", len(args)))
		}
		if filter.UploadedTo != nil {
			args = append(args, *filter.UploadedTo)
			where = append(where, fmt.Sprintf("f.uploaded_at <= $%d", len(args)))
		}
	}

	whereClause := strings.Join(where, " AND ")

	query := fmt.Sprintf(`
        select f.id, f.owner_id, f.blob_id, f.filename_original, f.filename_normalized,
               f.mime_declared, f.size_bytes_original, f.uploaded_at, f.is_deleted, f.tags, f.download_count,
               b.id, b.sha256, b.size_bytes, b.mime_detected, b.storage_key, b.ref_count, b.created_at
        from files f
        join file_blobs b on f.blob_id = b.id
        where %s
        order by f.uploaded_at desc
        limit 200
    `, whereClause)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	files := make([]FileWithBlob, 0)
	for rows.Next() {
		var rec FileRecord
		var blob FileBlob
		var tagsJSON []byte

		if err := rows.Scan(
			&rec.ID,
			&rec.OwnerID,
			&rec.BlobID,
			&rec.FilenameOriginal,
			&rec.FilenameNormalized,
			&rec.MimeDeclared,
			&rec.SizeBytesOriginal,
			&rec.UploadedAt,
			&rec.IsDeleted,
			&tagsJSON,
			&rec.DownloadCount,
			&blob.ID,
			&blob.Sha256,
			&blob.SizeBytes,
			&blob.MimeDetected,
			&blob.StorageKey,
			&blob.RefCount,
			&blob.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		if len(tagsJSON) > 0 {
			_ = json.Unmarshal(tagsJSON, &rec.Tags)
		} else {
			rec.Tags = []string{}
		}

		files = append(files, FileWithBlob{File: rec, Blob: blob})
	}

	countQuery := fmt.Sprintf(`
        select count(*)
        from files f
        join file_blobs b on f.blob_id = b.id
        where %s
    `, whereClause)

	argsCopy := make([]any, len(args))
	copy(argsCopy, args)

	var total int
	if err := p.QueryRow(ctx, countQuery, argsCopy...).Scan(&total); err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// ListPublicFiles returns publicly shared files (shares.visibility = 'PUBLIC' and not expired)
// with optional filters including uploader name/id. Results exclude deleted files.
func (p *Pool) ListPublicFiles(ctx context.Context, filter *FileFilter) ([]FileWithBlob, int, error) {
	args := []any{}
	// Only include files with a PUBLIC share that is not expired and has a valid token
	where := []string{
		"f.is_deleted = false",
		"s.visibility = 'PUBLIC'",
		"(s.expires_at is null or s.expires_at > now())",
		"(s.token is not null and s.token <> '')",
	}

	if filter != nil {
		if filter.Search != nil && *filter.Search != "" {
			args = append(args, "%"+strings.ToLower(*filter.Search)+"%")
			where = append(where, fmt.Sprintf("f.filename_normalized LIKE $%d", len(args)))
		}
		if len(filter.MimeTypes) > 0 {
			args = append(args, filter.MimeTypes)
			where = append(where, fmt.Sprintf("(coalesce(f.mime_declared, b.mime_detected) = ANY($%d))", len(args)))
		}
		if filter.MinSize != nil {
			args = append(args, *filter.MinSize)
			where = append(where, fmt.Sprintf("f.size_bytes_original >= $%d", len(args)))
		}
		if filter.MaxSize != nil {
			args = append(args, *filter.MaxSize)
			where = append(where, fmt.Sprintf("f.size_bytes_original <= $%d", len(args)))
		}
		if len(filter.Tags) > 0 {
			if tagsJSON, err := json.Marshal(filter.Tags); err == nil {
				args = append(args, string(tagsJSON))
				where = append(where, fmt.Sprintf("f.tags @> $%d", len(args)))
			}
		}
		if filter.UploadedFrom != nil {
			args = append(args, *filter.UploadedFrom)
			where = append(where, fmt.Sprintf("f.uploaded_at >= $%d", len(args)))
		}
		if filter.UploadedTo != nil {
			args = append(args, *filter.UploadedTo)
			where = append(where, fmt.Sprintf("f.uploaded_at <= $%d", len(args)))
		}
		if filter.UploaderName != nil && *filter.UploaderName != "" {
			args = append(args, "%"+strings.ToLower(*filter.UploaderName)+"%")
			where = append(where, fmt.Sprintf("(lower(u.name) LIKE $%d or lower(u.email) LIKE $%d)", len(args), len(args)))
		}
		if filter.UploaderID != nil {
			args = append(args, *filter.UploaderID)
			where = append(where, fmt.Sprintf("u.id = $%d", len(args)))
		}
	}

	whereClause := strings.Join(where, " AND ")

	query := fmt.Sprintf(`
		select f.id, f.owner_id, f.blob_id, f.filename_original, f.filename_normalized,
			   f.mime_declared, f.size_bytes_original, f.uploaded_at, f.is_deleted, f.tags, f.download_count,
			   b.id, b.sha256, b.size_bytes, b.mime_detected, b.storage_key, b.ref_count, b.created_at
		from shares s
		join files f on s.file_id = f.id
		join file_blobs b on f.blob_id = b.id
		join users u on u.id = f.owner_id
		where %s
		order by f.uploaded_at desc
		limit 200
	`, whereClause)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	files := make([]FileWithBlob, 0)
	for rows.Next() {
		var rec FileRecord
		var blob FileBlob
		var tagsJSON []byte
		if err := rows.Scan(
			&rec.ID,
			&rec.OwnerID,
			&rec.BlobID,
			&rec.FilenameOriginal,
			&rec.FilenameNormalized,
			&rec.MimeDeclared,
			&rec.SizeBytesOriginal,
			&rec.UploadedAt,
			&rec.IsDeleted,
			&tagsJSON,
			&rec.DownloadCount,
			&blob.ID,
			&blob.Sha256,
			&blob.SizeBytes,
			&blob.MimeDetected,
			&blob.StorageKey,
			&blob.RefCount,
			&blob.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if len(tagsJSON) > 0 {
			_ = json.Unmarshal(tagsJSON, &rec.Tags)
		} else {
			rec.Tags = []string{}
		}
		files = append(files, FileWithBlob{File: rec, Blob: blob})
	}

	countQuery := fmt.Sprintf(`
		select count(*)
		from shares s
		join files f on s.file_id = f.id
		join file_blobs b on f.blob_id = b.id
		join users u on u.id = f.owner_id
		where %s
	`, whereClause)

	argsCopy := make([]any, len(args))
	copy(argsCopy, args)

	var total int
	if err := p.QueryRow(ctx, countQuery, argsCopy...).Scan(&total); err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

func (p *Pool) MarkFileDeleted(ctx context.Context, fileID, ownerID uuid.UUID) (*FileRecord, error) {
	const stmt = `
        update files
        set is_deleted = true
        where id = $1 and owner_id = $2 and is_deleted = false
        returning id, blob_id, owner_id, filename_original, filename_normalized, mime_declared, size_bytes_original,
                  uploaded_at, tags, download_count
    `
	var rec FileRecord
	var tagsJSON []byte
	err := p.QueryRow(ctx, stmt, fileID, ownerID).Scan(
		&rec.ID,
		&rec.BlobID,
		&rec.OwnerID,
		&rec.FilenameOriginal,
		&rec.FilenameNormalized,
		&rec.MimeDeclared,
		&rec.SizeBytesOriginal,
		&rec.UploadedAt,
		&tagsJSON,
		&rec.DownloadCount,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if len(tagsJSON) > 0 {
		_ = json.Unmarshal(tagsJSON, &rec.Tags)
	} else {
		rec.Tags = []string{}
	}
	return &rec, nil
}

func (p *Pool) GetFileWithBlob(ctx context.Context, fileID, ownerID uuid.UUID) (*FileWithBlob, error) {
	const query = `
        select f.id, f.owner_id, f.blob_id, f.filename_original, f.filename_normalized,
               f.mime_declared, f.size_bytes_original, f.uploaded_at, f.is_deleted, f.tags, f.download_count,
               b.id, b.sha256, b.size_bytes, b.mime_detected, b.storage_key, b.ref_count, b.created_at
        from files f
        join file_blobs b on f.blob_id = b.id
        where f.id = $1 and f.owner_id = $2 and f.is_deleted = false
    `

	var rec FileRecord
	var blob FileBlob
	var tagsJSON []byte
	err := p.QueryRow(ctx, query, fileID, ownerID).Scan(
		&rec.ID,
		&rec.OwnerID,
		&rec.BlobID,
		&rec.FilenameOriginal,
		&rec.FilenameNormalized,
		&rec.MimeDeclared,
		&rec.SizeBytesOriginal,
		&rec.UploadedAt,
		&rec.IsDeleted,
		&tagsJSON,
		&rec.DownloadCount,
		&blob.ID,
		&blob.Sha256,
		&blob.SizeBytes,
		&blob.MimeDetected,
		&blob.StorageKey,
		&blob.RefCount,
		&blob.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if len(tagsJSON) > 0 {
		_ = json.Unmarshal(tagsJSON, &rec.Tags)
	} else {
		rec.Tags = []string{}
	}

	return &FileWithBlob{File: rec, Blob: blob}, nil
}

func (p *Pool) GetFileByShareToken(ctx context.Context, token string) (*FileRecord, *FileBlob, *ShareRecord, error) {
	const query = `
        select f.id, f.owner_id, f.blob_id, f.filename_original, f.filename_normalized,
               f.mime_declared, f.size_bytes_original, f.uploaded_at, f.tags, f.download_count,
               b.id, b.sha256, b.size_bytes, b.mime_detected, b.storage_key, b.ref_count, b.created_at,
               s.id, s.visibility, s.token, s.expires_at
        from shares s
        join files f on s.file_id = f.id
        join file_blobs b on f.blob_id = b.id
				where s.token = $1
					and (s.expires_at is null or s.expires_at > now())
          and f.is_deleted = false
    `

	var file FileRecord
	var blob FileBlob
	var share ShareRecord
	var tagsJSON []byte

	err := p.QueryRow(ctx, query, token).Scan(
		&file.ID,
		&file.OwnerID,
		&file.BlobID,
		&file.FilenameOriginal,
		&file.FilenameNormalized,
		&file.MimeDeclared,
		&file.SizeBytesOriginal,
		&file.UploadedAt,
		&tagsJSON,
		&file.DownloadCount,
		&blob.ID,
		&blob.Sha256,
		&blob.SizeBytes,
		&blob.MimeDetected,
		&blob.StorageKey,
		&blob.RefCount,
		&blob.CreatedAt,
		&share.ID,
		&share.Visibility,
		&share.Token,
		&share.ExpiresAt,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(tagsJSON) > 0 {
		_ = json.Unmarshal(tagsJSON, &file.Tags)
	} else {
		file.Tags = []string{}
	}

	return &file, &blob, &share, nil
}

func (p *Pool) IncrementDownload(ctx context.Context, fileID uuid.UUID) error {
	const stmt = `update files set download_count = download_count + 1 where id = $1`
	_, err := p.Exec(ctx, stmt, fileID)
	return err
}

func (p *Pool) UpsertShare(ctx context.Context, fileID uuid.UUID, visibility string, token *string, expires *time.Time) (*ShareRecord, error) {
	const stmt = `
        insert into shares (file_id, visibility, token, expires_at)
        values ($1, $2, $3, $4)
        on conflict (file_id)
            do update set visibility = excluded.visibility,
                          token = excluded.token,
                          expires_at = excluded.expires_at
        returning id, file_id, visibility, token, expires_at
    `
	var share ShareRecord
	err := p.QueryRow(ctx, stmt, fileID, visibility, token, expires).Scan(
		&share.ID,
		&share.FileID,
		&share.Visibility,
		&share.Token,
		&share.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return &share, nil
}

func (p *Pool) DeleteShare(ctx context.Context, fileID uuid.UUID) error {
	const stmt = `delete from shares where file_id = $1`
	_, err := p.Exec(ctx, stmt, fileID)
	return err
}

func (p *Pool) GetShareByFileID(ctx context.Context, fileID uuid.UUID) (*ShareRecord, error) {
	const query = `
        select id, file_id, visibility, token, expires_at
        from shares
        where file_id = $1
    `

	var share ShareRecord
	var token pgtype.Text
	var expires pgtype.Timestamptz

	err := p.QueryRow(ctx, query, fileID).Scan(&share.ID, &share.FileID, &share.Visibility, &token, &expires)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if token.Valid {
		share.Token = &token.String
	}
	if expires.Valid {
		t := expires.Time
		share.ExpiresAt = &t
	}

	return &share, nil
}

func (p *Pool) StorageUsage(ctx context.Context, ownerID uuid.UUID) (int64, int64, error) {
	const originalQuery = `
        select coalesce(sum(size_bytes_original), 0)
        from files
        where owner_id = $1 and is_deleted = false
    `
	var original int64
	if err := p.QueryRow(ctx, originalQuery, ownerID).Scan(&original); err != nil {
		return 0, 0, err
	}

	const dedupQuery = `
        select coalesce(sum(distinct b.size_bytes), 0)
        from files f
        join file_blobs b on f.blob_id = b.id
        where f.owner_id = $1 and f.is_deleted = false
    `
	var dedup int64
	if err := p.QueryRow(ctx, dedupQuery, ownerID).Scan(&dedup); err != nil {
		return 0, 0, err
	}

	return original, dedup, nil
}
