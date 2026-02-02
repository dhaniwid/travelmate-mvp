alter table trips
    add column if not exists status varchar(10) default 'DRAFT',
    add column if not exists updated_at timestamp with time zone default CURRENT_TIMESTAMP;