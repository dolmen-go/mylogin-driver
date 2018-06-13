// Small tool for testing the mylogin driver.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/dolmen-go/mylogin-driver/register"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: <conn-string> <SQL>")
	}
	db, err := sql.Open("mylogin", os.Args[1])
	if err != nil {
		log.Fatal("Open:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Ping:", err)
	}

	rows, err := db.Query(os.Args[2])
	if err != nil {
		log.Fatal("Query:", err)
	}
	defer rows.Close()

	var rowNum int64
	for rows.Next() {
		names, err := rows.Columns()
		if err != nil {
			log.Fatal("Columns:", err)
		}
		pvalues := make([]interface{}, len(names))
		for i := range pvalues {
			pvalues[i] = new(interface{})
		}
		if err = rows.Scan(pvalues...); err != nil {
			log.Fatalf("Scan %d: %v", rowNum, err)
		}
		for _, p := range pvalues {
			fmt.Print(*(p.(*interface{})), " ")
		}
		fmt.Println()
	}
	if err = rows.Err(); err != nil {
		log.Fatal("Next:", err)
	}
}
