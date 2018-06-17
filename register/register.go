/*
Package register provides auto-registration for mylogin-driver.Driver with name "mylogin".

Usage:

	import (
		"database/sql"
		_ "github.com/dolmen-go/mylogin-driver/register"
	)

	db := sql.Open("mylogin", "[filepath//]<section>/[<database>]")
*/
package register

import (
	"database/sql"

	mylogin "github.com/dolmen-go/mylogin-driver"
)

func init() {
	sql.Register("mylogin", mylogin.Driver{})
}
