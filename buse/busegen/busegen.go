package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
)

type BuseContext struct {
	Dir string
}

func main() {
	log.SetFlags(0)

	wd, err := os.Getwd()
	must(err)
	log.Printf("Working directory: (%s)", wd)

	bc := &BuseContext{
		Dir: wd,
	}

	must(bc.GenerateDocs())
	must(bc.GenerateGoCode())
}

func (bc *BuseContext) Task(task string) {
	log.Printf("")
	log.Printf("=========================")
	log.Printf(">> %s", task)
	log.Printf("=========================")
}

func (bc *BuseContext) ReadFile(file string) string {
	bs, err := ioutil.ReadFile(filepath.Join(bc.Dir, file))
	must(err)
	return string(bs)
}

func (bc *BuseContext) NewDoc(name string) *Doc {
	return &Doc{
		bc:   bc,
		name: name,
	}
}

func must(err error) {
	if err != nil {
		if se, ok := err.(*errors.Error); ok {
			log.Fatal(se.ErrorStack)
		} else {
			log.Fatal(se.Error)
		}
	}
}

//

type Doc struct {
	name string
	bc   *BuseContext

	doc string
	buf string
}

func (d *Doc) Load(doc string) {
	d.doc = doc
}

func (d *Doc) Line(msg string, args ...interface{}) {
	d.buf += fmt.Sprintf(msg, args...)
	d.buf += "\n"
}

func (d *Doc) Commit(name string) {
	if name == "" {
		d.doc = d.buf
	} else {
		d.doc = strings.Replace(d.doc, name, d.buf, 1)
	}
	d.buf = ""
}

func (b *Doc) Write() {
	bs := []byte(b.doc)
	dest := filepath.Join(b.bc.Dir, filepath.FromSlash(b.name))
	log.Printf("Writing (%s)...", dest)
	must(ioutil.WriteFile(dest, bs, 0644))
}
