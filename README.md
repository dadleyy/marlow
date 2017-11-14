<div style="text-align: center">
  <img src="https://s3.amazonaws.com/coverage.marlow.sizethree.cc/media/marlow.svg" width="65%" align="center"/>
</div>

---

Marlow is a code generation tool written in [golang] designed to create useful constructs that provide an ergonomic API
for interacting with a project's data persistence layer while maintaining strong compile time type safety assurance.

---- 

[![travis.img]][travis.url] 
[![codecov.img]][codecov.url]
[![report.img]][report.url]
[![godoc.img]][godoc.url]
[![tag.img]][tag.url]
[![commits.img]][commits.url]
[![awesome.img]][awesome.url]
[![generated-coverage.img]][generated-coverage.url]

---- 

### Objective &amp; Inspiration

Marlow was created to improve developer velocity on projects written in [golang] that interact with a data persistence
layer, like [mysql] or [postgres]. In other web application backend environments, these interfaces are usually
provided by an application framework's [ORM], like the [ActiveRecord] library used by [Rails].

For web applications leveraging the benefits of using [golang], it can be difficult to construct the equivalent
abstraction of their database that provides [crud] operations - especially one that is type-safe and [dry]. There are
several open source projects in the golang ecosystem who's goal is exactly that; [`gorm`], [`beego`], and  [`gorp`] to
name a [few][awesome-go]. Marlow differs from these other projects in its philosophy; rather than attempt to provide an
eloquent orm for your project at runtime, it generates a tailored solution at compile time.

### Usage

At its core, marlow simply reads a package's [field tags] and generates valid [golang] code. The `marlowc` executable 
can be installed & used directly via:

```
go get -u github.com/dadleyy/marlow/marlowc
marlowc -input=./examples/library/models -stdout=true
```

For a full list of options supported by the compiler refer to `marlowc -help`. The command line tool can also be used
as the executable target for golang's [`go generate`] command using `//go:generate` comment syntax:

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

### Generated Code &amp; Field Tag Configuration

The compiler parses the `marlow` field tag value using the `net/url` package's [`parseQuery`] function. This means that
each configuration option supported by marlow would end up in delimited by the ampersand (`&`) character where the key 
and value are separated by an equal sign (`=`). For example, a user record may look like:

```go
package models

type User struct {
  table string `marlow:"tableName=users"`
  ID    uint   `marlow:"column=id"`
  Name  string `marlow:"column=name"`
  Email string `marlow:"column=email"`
}
```

In this example, marlow would create the following `UserStore` interface in the `models` package:

```go
type UserStore interface {
	UpdateUserName(string, *UserBlueprint) (int64, error, string)
	FindUsers(*UserBlueprint) ([]*User, error)
	CountUsers(*UserBlueprint) (int, error)
	SelectIDs(*UserBlueprint) ([]uint, error)
	SelectEmails(*UserBlueprint) ([]string, error)
	CreateUsers(...User) (int64, error)
	UpdateUserEmail(string, *UserBlueprint) (int64, error, string)
	UpdateUserID(uint, *UserBlueprint) (int64, error, string)
	DeleteUsers(*UserBlueprint) (int64, error)
	SelectNames(*UserBlueprint) ([]string, error)
}
```

For every store that is generated, marlow will create a "blueprint" struct that defines a set of fields to be used for
querying against the database. In this example, the `UserBlueprint` generated for the store above would look like:

```go
type UserBlueprint struct {
	IDRange        []uint
	ID             []uint
	NameLike       []string
	Name           []string
	EmailLike      []string
	Email          []string
	Limit          int
	Offset         int
	OrderBy        string
	OrderDirection string
}
```

**Special `table` field**

If present, marlow will recognize the `table` field's `marlow` tag value as a container for developer specified 
overrides for default marlow assumptions about the table.

| Option | Description |
| :--- | :--- |
| `tableName` | The name of the table (marlow will assume a lowercased &amp; pluralized version of the struct name). |
| `dialect` | Specifying the `dialect` option determines the syntax used by marlow during db queries. See [supported-drivers](#supported-drivers) for more info. |
| `primaryKey` | Some dialects require that marlow knows of the table's primary key during transactions. If provided, this value will be used as the _column name_ by marlow. |
| `storeName` | The name of the store type that will be generated, defaults to `%sStore`, where `%s` is the name of the struct. |
| `blueprintName` | The name of the blueprint type that will be generated, defaults to `%sBlueprint`, where `%s` is the name of the struct. |
| `defaultLimit` | When using the queryable feature, this will be the default maximum number of records to load. |
| `blueprintRangeFieldSuffix` | A string that is added to numerical blueprint fields for range selections. Defults to `%sRange` where `%s` is the name of the field (e.g: `AuthorIDRange`). |
| `blueprintLikeFieldSuffix` | A string that is added to string/text blueprint fields for like selections. Defaults to `%sLike` where `%s` is the name of the field (e.g: `FirstNameLike`). |

**All other fields**

For every other field found on the source `struct`, marlow will use the following field tag values:

| Option | Description |
| :--- | :--- |
| `column` | This is the column that any raw sql generated will target when scanning/selecting/querying this field. |
| `autoIncrement` | If `true`, this flag will prevent marlow from generating sql during creation that would attempt to insert the value of the field for the column. |

#### Generated Coverage & Documentation

While everyone's generated marlow code will likely be unique, the [`examples/library`] application includes a
comprehensive test suite that demonstrates the features of marlow using 3 models - [`Author`], [`Book`] and [`Genre`].

Although generated documentation is not currently available (see [`GH-67`]), the html coverage reports for tagged 
builds can be found [here][generated-coverage.url]. There you will be able to see exactly what code is generated given
the models in the example app.

#### Supported Drivers

The follow is a list of officially support [`driver.Driver`] implementations supported by the generated marlow code. To request an additional driver, feel free to open up an [issue][issues].

| Driver Name | `dialect` Value |
| :--- | :--- |
| [`github.com/lib/pq`] | `postgres` | 
| [`github.com/mattn/go-sqlite3`] | none or `sql` | 
| [`github.com/go-sql-driver/mysql`] | none or `sql` | 


----

![logo][logo.img]

generated coverage badge provided by [gendry]

----

[preview]: https://s3.amazonaws.com/marlow-go/media/marlow.gif
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
[issues]: https://github.com/dadleyy/marlow/issues
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
[travis.img]: https://img.shields.io/travis/dadleyy/marlow/master.svg?style=flat-square
[travis.url]: https://travis-ci.org/dadleyy/marlow
[godoc.img]: http://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square
[godoc.url]: https://godoc.org/github.com/dadleyy/marlow/marlow
[field tags]: https://golang.org/pkg/reflect/#StructTag
[logo.img]: https://s3.amazonaws.com/coverage.marlow.sizethree.cc/media/marlow.svg
[commits.img]: https://img.shields.io/github/commits-since/dadleyy/marlow/latest.svg?style=flat-square
[commits.url]: https://github.com/dadleyy/marlow
[awesome.img]: https://img.shields.io/badge/%F0%9F%95%B6-awesome--go-443f5e.svg?colorA=c3a1bb&style=flat-square
[awesome.url]: https://awesome-go.com/#orm
[generated-coverage.img]: http://gendry.sizethree.cc/coverage/dadleyy/marlow.svg
[generated-coverage.url]: http://coverage.marlow.sizethree.cc.s3.amazonaws.com/latest/library.coverage.html
[gendry]: https://bitbucket.org/dadleyy/gendry
[`github.com/lib/pq`]: https://github.com/lib/pq
[`github.com/mattn/go-sqlite3`]: https://github.com/mattn/go-sqlite3
[`github.com/go-sql-driver/mysql`]: https://github.com/go-sql-driver/mysql
[`driver.Driver`]: https://golang.org/pkg/database/sql/driver/#Driver
[`examples/library`]: https://github.com/dadleyy/marlow/tree/master/examples/library
[`Author`]: https://github.com/dadleyy/marlow/blob/master/examples/library/models/author_test.go
[`Book`]: https://github.com/dadleyy/marlow/blob/master/examples/library/models/book_test.go
[`Genre`]: https://github.com/dadleyy/marlow/blob/master/examples/library/models/genre_test.go
[`GH-67`]: https://github.com/dadleyy/marlow/issues/67
