alter table shares
    add constraint shares_file_id_unique unique (file_id);
