// Package register provides auto-registration for mylogin-driver
package register

import (
	"database/sql"

	mylogin "github.com/dolmen-go/mylogin-driver"
)

func init() {
	sql.Register("mylogin", mylogin.Driver{})
}
