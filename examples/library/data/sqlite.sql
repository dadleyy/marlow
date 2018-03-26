drop table if exists authors;

create table authors (
  system_id INTEGER PRIMARY KEY,
  name TEXT,
  university_id INTEGER,
  rating REAL NOT NULL DEFAULT '100.00',
  flags INTEGER NOT NULL DEFAULT 0,
  birthday DATE NOT NULL,
  deceased DATE
);

drop table if exists books;

create table books (
  system_id INTEGER PRIMARY KEY,
  title TEXT,
  author INTEGER NOT NULL,
  series INTEGER,
  year_published INTEGER NOT NULL
);
