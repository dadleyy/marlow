# Marlow

Marlow is a code generation tool written in [golang] designed to generate useful constructs that provide an ergonomic
API for interacting with a project's data persistence layer while maintaining strong compile time type safety assurance.

---- 

[![travis.img]][travis.url] 
[![godoc.img]][godoc.url]
[![report.img]][report.url]
[![tag.img]][tag.url]
[![codecov.img]][codecov.url]

---- 

### Objective &amp; Inspiration

Marlow was created to improve developer velocity on projects written in [golang] that interact with a data persistence
layer, like [mysql] or [postgres]. In other programming languages, these interactions are usually provided by the 
project's [ORM], such as the [ActiveRecord] library used by [Rails].

For web applications leveraging the benefits of using [golang], it can be difficult to equivalent abstraction of their
database that provides [crud] operations - especially one that is type-safe and [dry]. This challenge has been 
approached by several popular libraries - [`gorm`], [`beego`], and [`gorp`] to name a [few][awesome-go]. 


### Useage

At its core, marlow simply reads a package's [field tags] and generates valid [golang] code. The marlow compiler can
be installed & used directly via:

```
go get -u github.com/dadleyy/marlow/marlowc
marlowc -input=./examples/library/models -stdout=true
```

For a full list of options supported by the compiler refer to `marlowc -help`. The compiler can also be used to compile
using the golang [`go generate`] command using the generator tool's comment syntax:

```go
package models

//go:generate marlow -input=book.go

type Book struct {
  ID       string `marlow="column=id"`
  AuthorID string `marlow="column=author_id"`
}
```

The generated files will live in the same directory as their source counterparts, with an optional suffix to 
distinguish them (useful if a project is using [`make`] to manage the build pipeline). In general it is encouraged that
the generated files **are not committed to your project's revision control**; the source should *always* be generated
immediately before the rest of the package's source code is compiled.

### Field Tag Configuration

The compiler parses the `marlow` field tag value using the `net/url` package's [`parseQuery`] function. This means that
each configuration option supported by marlow would end up in delimited by the ampersand (`&`) character where the key 
and value are separated by an equal sign (`=`). For example, a user record may look like:

```go
package Model

type User struct {
  table string `marlow:"name=users&timestamps=[created_at, deleted_at]"`
  ID    uint   `marlow:"column=id&primary=true"`
  Name  string `marlow:"column=name"`
  Email string `marlow:"column=name&unique=true"`
}
```

In this example, marlow would create golang code that would look (not exactly) like this:

```go
func (s *UserStore) FindUsers(query *UserQuery) ([]*User, error) {
  out := make([]*User, 0)
  // ...
  _rows, e := s.DB.Query(_generatedSQL) // e.g "SELECT id, name FROM users ..."
  // ... 

  for _rows.Next() {
    var _u User

    if e := _rows.Scan(&u.Id, &u.Name); e != nil {
      return e
    }

    out = append(out, &_u)
  }

  // ...
}
```

**Special `table` field**

If present, marlow will recognize the `table` field's `marlow` tag value as a container for developer specified 
overrides for default marlow assumptions about the table.

| Option | Description |
| :--- | :--- |
| `name` | The name of the table (marlow will assume a lowercased &amp; pluralized version of the struct name). |
| `timestamps` | Array indicating presence of `created_at`, `updated_at`, and `destroyed_at` columns. |

**All other fields**

| Option | Description |
| :--- | :--- |
| `column` | This is the column that any raw sql generated will target when scanning/selecting/querying this field. |
| `primary` | Lets marlow know that this is the primary key of the table. |

[`ParseQuery`]: https://golang.org/pkg/net/url/#ParseQuery
[`make`]: https://www.gnu.org/software/make/
[`go generate`]: https://blog.golang.org/generate
[awesome-go]: https://github.com/avelino/awesome-go#orm
[`gorm`]: https://github.com/jinzhu/gorm
[`gorp`]: https://github.com/go-gorp/gorp
[`beego`]: https://github.com/astaxie/beego/tree/master/orm
[crud]: https://en.wikipedia.org/wiki/Create,_read,_update_and_delete
[dry]: https://en.wikipedia.org/wiki/Don%27t_repeat_yourself
[Rails]: http://rubyonrails.org
[ActiveRecord]: http://guides.rubyonrails.org/active_record_basics.html
[ORM]: https://en.wikipedia.org/wiki/Object-relational_mapping
[mysql]: https://www.mysql.com
[postgres]: https://www.postgresql.org
[codecov.img]: https://img.shields.io/codecov/c/github/dadleyy/marlow.svg?style=flat-square
[codecov.url]: https://codecov.io/gh/dadleyy/marlow
[tag.img]: https://img.shields.io/github/tag/dadleyy/marlow.svg?style=flat-square
[tag.url]: https://github.com/dadleyy/marlow/releases
[struct field tags]: https://golang.org/ref/spec#Tag
[golang]: https://golang.org
[report.img]: https://goreportcard.com/badge/github.com/dadleyy/marlow?style=flat-square
[report.url]: https://goreportcard.com/report/github.com/dadleyy/marlow
[travis.img]: https://img.shields.io/travis/dadleyy/marlow.svg?style=flat-square
[travis.url]: https://travis-ci.org/dadleyy/marlow
[godoc.img]: http://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square
[godoc.url]: https://godoc.org/github.com/dadleyy/marlow/marlow
