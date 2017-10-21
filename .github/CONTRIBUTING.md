### Contributing to [Marlow]

Thank you for your interest in [marlow]! This is an open source project; pull requests and issues are welcome from all.

The codebase's [travis ci build][travis] is set up to mimic the same check suite as the [goreportcard] website, so
every PR will be subject to those standards, as well as maintaining the code coverage level configured at [codecov.io].

If you are interested in contributing, please familiarize yourself with the automation tools the project has configured
for continuous integration _before_ opening a pull request; while not absolutely necessary, it will save you some
surprises when you do open a pull request that could've been mitigated locally.

### Pull requests

When opening a pull request, be sure to use the `PULL_REQUEST_TEMPLATE.md` file in the `.github` directory (github should automatically fill your PR description with the contents of that file) as a template for your PR's description. The first table (underneath "Notable Changes") should explicitly call out any important line changes, as well as the github issue number and/or general reason for the change.

The second table should be used to identify other developers that should review the code. This includes both the primary reviewer - using a `@mention` next to the :tophat: row - and any other developers that may be interested in the diff next to the :paperclip:. For example:

> | :tophat: | @dadleyy |
> | :--- | :--- |
> | :paperclip: | @sizethree/golang |

would indicate that @dadleyy is primarily responsible for reviewing the code, while the @sizethree/golang team would be interested in it.


[Marlow]: https://github.com/dadleyy/marlow
[travis]: https://travis-ci.org/dadleyy/marlow
[goreportcard]: https://goreportcard.com
[codecov.io]: https://codecov.io/gh/dadleyy/marlow
