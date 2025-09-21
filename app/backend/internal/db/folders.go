package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Folder struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	ParentID  *uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *Pool) CreateFolder(ctx context.Context, ownerID uuid.UUID, name string, parentID *uuid.UUID) (*Folder, error) {
	const stmt = `
        insert into folders (owner_id, parent_id, name)
        values ($1, $2, $3)
        returning id, owner_id, parent_id, name, created_at, updated_at
    `

	var folder Folder
	var parent pgtype.UUID

	err := p.QueryRow(ctx, stmt, ownerID, parentID, name).Scan(
		&folder.ID,
		&folder.OwnerID,
		&parent,
		&folder.Name,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	parentPtr, err := uuidPtrFromPG(parent)
	if err != nil {
		return nil, err
	}
	folder.ParentID = parentPtr

	return &folder, nil
}

func (p *Pool) RenameFolder(ctx context.Context, folderID, ownerID uuid.UUID, name string) (*Folder, error) {
	const stmt = `
        update folders
        set name = $3, updated_at = now()
        where id = $1 and owner_id = $2
        returning id, owner_id, parent_id, name, created_at, updated_at
    `

	var folder Folder
	var parent pgtype.UUID

	err := p.QueryRow(ctx, stmt, folderID, ownerID, name).Scan(
		&folder.ID,
		&folder.OwnerID,
		&parent,
		&folder.Name,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	parentPtr, err := uuidPtrFromPG(parent)
	if err != nil {
		return nil, err
	}
	folder.ParentID = parentPtr

	return &folder, nil
}

func (p *Pool) DeleteFolder(ctx context.Context, folderID, ownerID uuid.UUID) (bool, error) {
	const stmt = `delete from folders where id = $1 and owner_id = $2`
	tag, err := p.Exec(ctx, stmt, folderID, ownerID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (p *Pool) GetFolderByID(ctx context.Context, folderID uuid.UUID) (*Folder, error) {
	const query = `
        select id, owner_id, parent_id, name, created_at, updated_at
        from folders
        where id = $1
    `

	var folder Folder
	var parent pgtype.UUID

	err := p.QueryRow(ctx, query, folderID).Scan(
		&folder.ID,
		&folder.OwnerID,
		&parent,
		&folder.Name,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	parentPtr, err := uuidPtrFromPG(parent)
	if err != nil {
		return nil, err
	}
	folder.ParentID = parentPtr

	return &folder, nil
}

func (p *Pool) ListFolders(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID) ([]Folder, error) {
	const query = `
        select id, owner_id, parent_id, name, created_at, updated_at
        from folders
        where owner_id = $1
          and ( ($2::uuid is null and parent_id is null) or ($2::uuid is not null and parent_id = $2) )
        order by lower(name)
    `

	rows, err := p.Query(ctx, query, ownerID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	folders := make([]Folder, 0)
	for rows.Next() {
		var folder Folder
		var parent pgtype.UUID
		if err := rows.Scan(&folder.ID, &folder.OwnerID, &parent, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return nil, err
		}
		parentPtr, err := uuidPtrFromPG(parent)
		if err != nil {
			return nil, err
		}
		folder.ParentID = parentPtr
		folders = append(folders, folder)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return folders, nil
}
func (p *Pool) ListFolderTree(ctx context.Context, ownerID, rootID uuid.UUID) ([]Folder, error) {
	const query = `
        with recursive folder_tree as (
            select id, owner_id, parent_id, name, created_at, updated_at
            from folders
            where id = $2 and owner_id = $1
            union all
            select f.id, f.owner_id, f.parent_id, f.name, f.created_at, f.updated_at
            from folders f
            join folder_tree ft on f.parent_id = ft.id
        )
        select id, owner_id, parent_id, name, created_at, updated_at
        from folder_tree
    `

	rows, err := p.Query(ctx, query, ownerID, rootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	folders := make([]Folder, 0)
	for rows.Next() {
		var folder Folder
		var parent pgtype.UUID
		if err := rows.Scan(&folder.ID, &folder.OwnerID, &parent, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return nil, err
		}
		parentPtr, err := uuidPtrFromPG(parent)
		if err != nil {
			return nil, err
		}
		folder.ParentID = parentPtr
		folders = append(folders, folder)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return folders, nil
}
