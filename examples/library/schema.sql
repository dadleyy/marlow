drop table if exists authors;

create table authors (
  id integer not null primary key,
  name text,
  university_id integer
);

drop table if exists books;

create table books (
  id integer not null primary key,
  title text,
  author_id integer not null,
  series_id integer,
  page_count integer not null
);
