/*

Connection string syntax:

    [<path>//]<section>/[<database>]
*/
package mylogindriver

import (
	"context"
	"database/sql/driver"
	"errors"
	"path/filepath"
	"strings"

	"github.com/go-sql-driver/mysql"

	"github.com/dolmen-go/mylogin"
)

type Driver struct{}

var (
	errInvalidSyntax = errors.New("invalid connection string")
)

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
	if i >= 0 {
		options = dbName[i:]
		dbName = dbName[:i]
	}
	return connector(login.DSN() + dbName + options), nil
}

func (drv Driver) Open(name string) (driver.Conn, error) {
	cnt, err := drv.OpenConnector(name)
	if err != nil {
		return nil, err
	}
	return cnt.Connect(context.TODO())
}

type connector string

func (cnt connector) Driver() driver.Driver {
	return Driver{}
}

func (cnt connector) Connect(context.Context) (driver.Conn, error) {
	return mysql.MySQLDriver{}.Open(string(cnt))
}
