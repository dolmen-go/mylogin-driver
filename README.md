# mylogin-driver - Go database/sql driver for MySQL loading credentials from ~/.mylogin.cnf

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/dolmen-go/mylogin-driver)
[![Travis-CI](https://img.shields.io/travis/dolmen-go/mylogin-driver.svg)](https://travis-ci.org/dolmen-go/mylogin-driver)
[![Go Report Card](https://goreportcard.com/badge/github.com/dolmen-go/mylogin-driver)](https://goreportcard.com/report/github.com/dolmen-go/mylogin-driver)

`mylogin-driver` provides a MySQL driver for
[`database/sql`](https://golang.org/pkg/database/sql/).
This is just a wrapper around
[`github.com/go-sql-driver/mysql`](https://github.com/go-sql-driver/mysql) with
a different connection string syntax that allows to read server address and
credentials from `~/.mylogin.cnf`.

About mylogin.cnf:

- <https://dev.mysql.com/doc/refman/8.0/en/mysql-config-editor.html>
- <https://dev.mysql.com/doc/mysql-utilities/1.5/en/mysql-utils-intro-connspec-mylogin.cnf.html>

See also package [`github.com/dolmen-go/mylogin`](https://pkg.go.dev/github.com/dolmen-go/mylogin)
that provides low-level access to `~/.mylogin.cnf` reading and writing.

## Usage



```go
import (
    "database/sql"
    _ "github.com/dolmen-go/mylogin-driver/register"
)

db, err := sql.Open("mylogin", "[filepath//]<section>/[<database>]")
```

## Example

Here is an example with the
[public MySQL server provided by ensembl.org](https://www.ensembl.org/info/data/mysql.html).

Create MySQL credentials in a section named `client_ensembl` in `~/.mylogin.cnf`:

```sh
$ mysql_config_editor set -G client_ensembl -h ensembldb.ensembl.org -P 5306 -u anonymous -p
$ cat >main.go <<EOF
package main

import (
    "database/sql"
    "fmt"
    _ "github.com/dolmen-go/mylogin-driver/register"
)

func main() {
    db, _ := sql.Open("mylogin", "client_ensembl/")
    var s string
    _ = db.QueryRow("SELECT NOW()").Scan(&s)
    fmt.Println(s)
}
EOF
$ go run main.go
```

## License

Copyright 2018 Olivier Mengué

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   <http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
