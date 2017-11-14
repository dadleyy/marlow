drop table if exists genres;

create table genres (
  id SERIAL,
  name TEXT,
  parent_id INTEGER
);
