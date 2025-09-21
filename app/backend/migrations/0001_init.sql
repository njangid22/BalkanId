create extension if not exists "pgcrypto";

create table if not exists users (
  id uuid primary key default gen_random_uuid(),
  email text unique not null,
  name text,
  role text not null default 'USER',
  quota_bytes bigint not null default 10485760,
  created_at timestamptz not null default now()
);

create table if not exists file_blobs (
  id uuid primary key default gen_random_uuid(),
  sha256 text not null unique,
  size_bytes bigint not null,
  mime_detected text not null,
  storage_key text not null,
  ref_count integer not null default 1,
  created_at timestamptz not null default now()
);

create table if not exists files (
  id uuid primary key default gen_random_uuid(),
  owner_id uuid not null references users(id) on delete cascade,
  blob_id uuid not null references file_blobs(id),
  filename_original text not null,
  filename_normalized text not null,
  mime_declared text,
  size_bytes_original bigint not null,
  uploaded_at timestamptz not null default now(),
  is_deleted boolean not null default false,
  tags jsonb not null default '[]'::jsonb,
  download_count bigint not null default 0
);

create index if not exists idx_files_name on files (lower(filename_normalized));
create index if not exists idx_files_uploaded_at on files (uploaded_at);
create index if not exists idx_files_size on files (size_bytes_original);
create index if not exists idx_files_tags on files using gin (tags);

create table if not exists shares (
  id uuid primary key default gen_random_uuid(),
  file_id uuid references files(id) on delete cascade,
  visibility text not null default 'private',
  token text unique,
  expires_at timestamptz
);

create table if not exists audit_logs (
  id uuid primary key default gen_random_uuid(),
  actor_id uuid references users(id),
  action text not null,
  entity_type text not null,
  entity_id uuid,
  at timestamptz not null default now(),
  metadata jsonb
);
