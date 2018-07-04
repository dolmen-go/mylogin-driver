// Small tool for testing the mylogin driver.
package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/dolmen-go/flagx"

	_ "github.com/dolmen-go/mylogin-driver/register"
)

type layout interface {
	writeHeader(columns []string)
	writeRow(row []interface{}) error
	writeFooter()
}

type baseLayout struct {
	w io.Writer
}

func (baseLayout) writeHeader([]string) {}

func (baseLayout) writeRow(row []interface{}) error {
	for _, c := range row {
		fmt.Print(c, " ")
	}
	fmt.Println()
	return nil
}

func (baseLayout) writeFooter() {}

type jsonLayout struct {
	baseLayout
	enc   *json.Encoder
	first bool
}

func newJSON(w io.Writer) (layout, error) {
	return &jsonLayout{baseLayout{w: w}, json.NewEncoder(w), true}, nil
}

func (j *jsonLayout) writeHeader([]string) {
	j.baseLayout.w.Write([]byte("[\n"))
}

func (j *jsonLayout) writeRow(row []interface{}) error {
	if j.first {
		j.baseLayout.w.Write([]byte{' '})
		j.first = false
	} else {
		j.baseLayout.w.Write([]byte{','})
	}
	for i, c := range row {
		if bin, isBin := c.([]byte); isBin {
			row[i] = string(bin)
		}
	}
	err := j.enc.Encode(row)
	if err != nil {
		return err
	}
	//j.baseLayout.w.Write([]byte{'\n'})
	return nil
}

func (j *jsonLayout) writeFooter() {
	j.baseLayout.w.Write([]byte("]\n"))
}

type jsonObjectLayout struct {
	baseLayout
	first bool
	keys  [][]byte
}

func newJSONObject(w io.Writer) (layout, error) {
	return &jsonObjectLayout{baseLayout{w: w}, true, nil}, nil
}

func (j *jsonObjectLayout) writeHeader(names []string) {
	if len(names) > 0 {
		keys := make([][]byte, len(names))
		for i, name := range names {
			enc, _ := json.Marshal(name)
			key := make([]byte, 0, 2+len(enc))
			if i > 0 {
				key = append(key, ',')
			}
			keys[i] = append(append(key, enc...), ':')
		}
		j.keys = keys
	}
	j.baseLayout.w.Write([]byte("[\n"))
}

func (j *jsonObjectLayout) writeRow(row []interface{}) error {
	if j.first {
		j.baseLayout.w.Write([]byte{' ', '{'})
		j.first = false
	} else {
		j.baseLayout.w.Write([]byte{',', '{'})
	}
	for i, c := range row {
		if bin, isBin := c.([]byte); isBin {
			c = string(bin)
		}
		enc, err := json.Marshal(c)
		if err != nil {
			return err
		}
		j.baseLayout.w.Write(j.keys[i])
		j.baseLayout.w.Write(enc)
	}
	j.baseLayout.w.Write([]byte{'}', '\n'})
	return nil
}

func (j *jsonObjectLayout) writeFooter() {
	j.baseLayout.w.Write([]byte("]\n"))
}

var output layout = &baseLayout{w: os.Stdout}

func declareLayout(name string, help string, builder func(w io.Writer) (layout, error)) {
	flag.Var(flagx.BoolFunc(func(b bool) error {
		if !b {
			return errors.New("can't disable a layout")
		}
		l, err := builder(os.Stdout)
		if err != nil {
			return err
		}
		output = l
		return nil
	}), name, help)
}

func main() {
	declareLayout("json-array", "JSON output: each row is an array", newJSON)
	declareLayout("json-object", "JSON output: each row is an object with column names as keys", newJSONObject)

	flag.Parse()

	if flag.NArg() < 2 {
		log.Println(flag.NArg())
		log.Fatal("usage: [options...] <conn-string> <SQL> [args...]")
	}
	db, err := sql.Open("mylogin", flag.Arg(0))
	if err != nil {
		log.Fatal("Open:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Ping:", err)
	}

	var args []interface{}
	if flag.NArg() > 2 {
		args = make([]interface{}, flag.NArg()-2)
		for i, a := range flag.Args()[2:] {
			args[i] = a
		}
	}

	// Force a prepared statement to avoid MySQL "text mode" that hides column type
	stmt, err := db.Prepare(flag.Arg(1))
	if err != nil {
		log.Fatal("Prepare:", err)
	}
	defer stmt.Close()
	rows, err := stmt.Query(args...)
	if err != nil {
		log.Fatal("Exec:", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err = rows.Err(); err != nil {
			log.Fatal("Next:", err)
		}

		output.writeHeader(nil)
		output.writeFooter()
		return
	}

	names, err := rows.Columns()
	if err != nil {
		log.Fatal("Columns:", err)
	}
	output.writeHeader(names)

	rowNum := int64(1)
	for {
		row := make([]interface{}, len(names))
		pvalues := make([]interface{}, len(names))
		for i := range pvalues {
			pvalues[i] = &row[i]
		}
		if err = rows.Scan(pvalues...); err != nil {
			log.Fatalf("Scan %d: %v", rowNum, err)
		}
		if err = output.writeRow(row); err != nil {
			rows.Close()
			log.Fatal(err)
		}
		if !rows.Next() {
			break
		}
		rowNum++
	}
	if err = rows.Err(); err != nil {
		log.Fatal("Next:", err)
	}
	output.writeFooter()
}
