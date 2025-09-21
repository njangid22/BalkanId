create table if not exists folders (
    id uuid primary key default gen_random_uuid(),
    owner_id uuid not null references users(id) on delete cascade,
    parent_id uuid references folders(id) on delete cascade,
    name text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists idx_folders_owner on folders(owner_id);
create index if not exists idx_folders_parent on folders(parent_id);

create unique index if not exists uq_folders_owner_parent_name
    on folders(owner_id, coalesce(parent_id, '00000000-0000-0000-0000-000000000000'::uuid), lower(name));

alter table files
    add column if not exists folder_id uuid references folders(id) on delete set null;

create index if not exists idx_files_folder on files(folder_id);

do $$
begin
    if exists (
        select 1 from information_schema.columns
        where table_name = 'shares' and column_name = 'file_id'
    ) then
        alter table shares drop constraint if exists shares_file_id_unique;
        alter table shares drop constraint if exists shares_file_id_fkey;

        alter table shares add column if not exists target_type text;
        alter table shares add column if not exists target_id uuid;

        update shares set target_type = 'FILE', target_id = file_id where target_type is null;

        alter table shares alter column target_type set not null;
        alter table shares alter column target_id set not null;

        alter table shares drop column file_id;
    end if;
end
$$;

alter table shares add constraint shares_target_type_check
    check (target_type in ('FILE', 'FOLDER'));

create unique index if not exists shares_target_unique
    on shares(target_type, target_id);
