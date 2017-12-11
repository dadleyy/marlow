drop table if exists genres;

create table genres (
  id SERIAL,
  name TEXT,
  parent_id INTEGER
);

drop table if exists multi_auto;

create table multi_auto (
  id SERIAL,
  status TEXT DEFAULT 'pending' NOT NULL,
  name TEXT
);
