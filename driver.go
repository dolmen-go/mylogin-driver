/*
Package mylogindriver provides a database/sql driver for MySQL using credentials stored in ~/.mylogin.cnf.

Connection string syntax:

    [<filepath>//]<section>/[<database>][?<options>]

Default filepath is $HOME/.mylogin.cnf or $MYSQL_TEST_LOGIN_FILE. See https://pkg.go.dev/github.com/dolmen-go/mylogin/#DefaultFile.

About mylogin.cnf:
    https://dev.mysql.com/doc/refman/8.0/en/mysql-config-editor.html
    https://dev.mysql.com/doc/mysql-utilities/1.5/en/mysql-utils-intro-connspec-mylogin.cnf.html

mylogindriver is opiniated. The following options are set if not explicitely given a value:

    tls=preferred      Enable TLS if the MySQL server supports it. Note that TLS certificates are NOT verified.

A package that auto-registers the driver is provided in https://pkg.go.dev/github.com/dolmen-go/mylogin-driver/register/.
*/
package mylogindriver

import (
	"context"
	"database/sql/driver"
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-sql-driver/mysql"

	"github.com/dolmen-go/mylogin"
)

// Driver is a database/sql driver.
//
// It implements interfaces driver.Driver and driver.DriverContext.
type Driver struct {
	private struct{} // just to make the internal representation secret.
}

var (
	errInvalidSyntax = errors.New("invalid connection string")
)

// OpenConnector implements interface database/sql/driver.DriverContext.
func (drv Driver) OpenConnector(name string) (driver.Connector, error) {
	var path string
	i := strings.Index(name, "//")
	if i >= 0 {
		path = filepath.FromSlash(name[:i])
		name = name[i+2:]
	}
	if len(path) == 0 {
		path = mylogin.DefaultFile()
	}
	i = strings.IndexByte(name, '/')
	if i < 1 {
		return nil, errInvalidSyntax
	}
	login, err := mylogin.ReadLogin(path, []string{name[:i], mylogin.DefaultSection})
	if err != nil {
		return nil, err
	}
	var options string
	dbName := name[i+1:]
	i = strings.IndexByte(dbName, '?')
	if i >= 0 && i+1 < len(dbName) {
		options = dbName[i+1:]
		dbName = dbName[:i]
		if q, err := url.ParseQuery(options); err == nil {
			changed := false
			if q.Get("tls") == "" {
				q.Set("tls", "preferred")
				changed = true
			}
			if changed {
				options = q.Encode()
			}
		}
	} else {
		options = "tls=preferred"
	}
	c, err := mysql.MySQLDriver{}.OpenConnector(login.DSN() + dbName + "?" + options)
	if err != nil {
		return nil, err
	}
	return connector{c: c}, nil
}

// Open implements interface database/sql/driver.Driver.
func (drv Driver) Open(name string) (driver.Conn, error) {
	cnt, err := drv.OpenConnector(name)
	if err != nil {
		return nil, err
	}
	return cnt.Connect(context.TODO())
}

// connector implements interface database/sql/driver.Connector.
type connector struct {
	c driver.Connector
}

// Driver implements interface database/sql/driver.Connector.
func (cnt connector) Driver() driver.Driver {
	return Driver{}
}

// Connect implements interface database/sql/driver.Connector.
func (cnt connector) Connect(ctx context.Context) (driver.Conn, error) {
	return cnt.c.Connect(ctx)
}
