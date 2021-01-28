// Small tool for testing the mylogin driver.
//
// Install
//    go get -u github.com/dolmen-go/mylogin-driver
//    go install github.com/dolmen-go/mylogin-driver/cmd/mylogin-query
package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/dolmen-go/flagx"
	"golang.org/x/text/encoding/unicode"

	_ "github.com/dolmen-go/mylogin-driver/register"
)

type layout interface {
	writeHeader(columns []string) error
	writeRow(row []interface{}) error
	writeFooter() error
}

type baseLayout struct {
	w io.Writer
}

func (baseLayout) writeHeader([]string) error {
	return nil
}

func (baseLayout) writeRow(row []interface{}) error {
	for _, c := range row {
		fmt.Print(c, " ")
	}
	fmt.Println()
	return nil
}

func (baseLayout) writeFooter() error {
	return nil
}

type csvLayout struct {
	baseLayout
	w   *csv.Writer
	row []string
}

func newCSV(w io.Writer) (layout, error) {
	return &csvLayout{w: csv.NewWriter(w)}, nil
}

func (l *csvLayout) writeHeader(columns []string) error {
	if columns == nil {
		return nil
	}
	return l.w.Write(columns)
}

func (l *csvLayout) writeRow(row []interface{}) error {
	if l.row == nil {
		l.row = make([]string, len(row))
	}
	for i, c := range row {
		switch c := c.(type) {
		case []byte:
			l.row[i] = string(c)
		default:
			l.row[i] = fmt.Sprint(c)
		}
	}
	return l.w.Write(l.row)
}

func (l *csvLayout) writeFooter() error {
	l.w.Flush()
	return l.w.Error()
}

func newCSVExcel(w io.Writer) (layout, error) {
	const sep rune = ';'

	/*
		const utf8BOM = "\xEF\xBB\xBF"
		_, err := w.Write([]byte(utf8BOM + "sep=" + string(sep) + "\n"))
	*/
	w = unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Writer(w)
	_, err := w.Write([]byte("sep=" + string(sep) + "\n"))

	if err != nil {
		return nil, err
	}
	l, err := newCSV(w)
	l.(*csvLayout).w.Comma = sep
	return l, err
}

type jsonLayout struct {
	baseLayout
	enc   *json.Encoder
	first bool
}

func newJSON(w io.Writer) (layout, error) {
	return &jsonLayout{baseLayout{w: w}, json.NewEncoder(w), true}, nil
}

func (j *jsonLayout) writeHeader([]string) error {
	_, err := j.baseLayout.w.Write([]byte("[\n"))
	return err
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

func (j *jsonLayout) writeFooter() error {
	_, err := j.baseLayout.w.Write([]byte("]\n"))
	return err
}

type jsonObjectLayout struct {
	baseLayout
	first bool
	keys  [][]byte
}

func newJSONObject(w io.Writer) (layout, error) {
	return &jsonObjectLayout{baseLayout{w: w}, true, nil}, nil
}

func (j *jsonObjectLayout) writeHeader(names []string) error {
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
	_, err := j.baseLayout.w.Write([]byte("[\n"))
	return err
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

func (j *jsonObjectLayout) writeFooter() error {
	_, err := j.baseLayout.w.Write([]byte("]\n"))
	return err
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
	declareLayout("csv", "CSV output", newCSV)
	declareLayout("csv-Excel", "CSV output, encoded as UTF-16LE with BOM and special Excel header", newCSVExcel)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [options...] <mylogin-section>/<database>[?<options>] <SQL> [args...]\n\noptions:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 2 || flag.Arg(0) == "" || flag.Arg(1) == "" {
		flag.Usage()
		os.Exit(1)
	}
	db, err := sql.Open("mylogin", flag.Arg(0))
	if err != nil {
		log.Fatal("Open:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Ping:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// interrupt context with SIGTERM (CTRL+C)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		cancel()
	}()

	var args []interface{}
	if flag.NArg() > 2 {
		args = make([]interface{}, flag.NArg()-2)
		for i, a := range flag.Args()[2:] {
			args[i] = a
		}
	}

	// Force a prepared statement to avoid MySQL "text mode" that hides column type
	stmt, err := db.PrepareContext(ctx, flag.Arg(1))
	if err != nil {
		log.Fatal("Prepare:", err)
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		log.Fatal("Exec:", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err = rows.Err(); err != nil {
			log.Fatal("Next:", err)
		}

		if err = output.writeHeader(nil); err != nil {
			log.Fatal(err)
		}
		if err = output.writeFooter(); err != nil {
			log.Fatal(err)
		}
		return
	}

	names, err := rows.Columns()
	if err != nil {
		log.Fatal("Columns:", err)
	}
	if err = output.writeHeader(names); err != nil {
		log.Fatal(err)
	}

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
	if err = output.writeFooter(); err != nil {
		log.Fatal(err)
	}
}
