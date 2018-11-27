package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Doc struct {
	name string
	gc   *GenerousContext

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
	dest := b.name
	log.Printf("Writing (%s)...", dest)
	must(os.MkdirAll(filepath.Dir(dest), 0755))
	must(ioutil.WriteFile(dest, bs, 0644))
}
