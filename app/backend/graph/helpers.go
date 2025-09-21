package graph

import (
	"time"
	"vault/graph/model"
	"vault/internal/db"
)

func mapUser(u db.User) *model.User {
	return &model.User{
		ID:         u.ID.String(),
		Email:      u.Email,
		Name:       u.Name,
		Role:       model.Role(u.Role),
		QuotaBytes: int(u.QuotaBytes),
		CreatedAt:  u.CreatedAt,
	}
}

func mapFile(rec db.FileRecord, blob db.FileBlob, owner *model.User, deduped bool) *model.File {
	var detected *string
	if blob.MimeDetected != "" {
		md := blob.MimeDetected
		detected = &md
	}
	return &model.File{
		ID:                rec.ID.String(),
		Owner:             owner,
		FilenameOriginal:  rec.FilenameOriginal,
		SizeBytesOriginal: int(rec.SizeBytesOriginal),
		MimeDeclared:      rec.MimeDeclared,
		MimeDetected:      detected,
		UploadedAt:        rec.UploadedAt,
		DownloadCount:     int(rec.DownloadCount),
		Deduped:           deduped,
		Tags:              rec.Tags,
	}
}

func mapShare(s db.ShareRecord, file *model.File) *model.Share {
	return &model.Share{
		ID:         s.ID.String(),
		File:       file,
		Visibility: model.ShareVisibility(s.Visibility),
		Token:      s.Token,
		ExpiresAt:  s.ExpiresAt,
	}
}

func toTimePtr(t *time.Time) *time.Time { return t }
