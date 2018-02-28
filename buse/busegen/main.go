package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-errors/errors"
)

type BuseContext struct {
	Dir string
}

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Printf("Usage: busegen (godocs|ts [OUT])")
		log.Printf("  - godocs: generate directly in the $GOPATH tree")
		log.Printf("  - ts: give a target path to generate")
		os.Exit(1)
	}
	mode := os.Args[1]

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Fatalf("$GOPATH must be set")
	}

	baseDir := filepath.Join(gopath, "src", "github.com", "itchio", "butler", "buse", "busegen")
	_, err := os.Stat(baseDir)
	must(err)
	log.Printf("Base dir: (%s)", baseDir)

	bc := &BuseContext{
		Dir: baseDir,
	}

	switch mode {
	case "godocs":
		must(bc.GenerateDocs())
		must(bc.GenerateGoCode())
		must(bc.GenerateSpec())
	case "ts":
		if len(os.Args) < 2 {
			log.Printf("busegen ts: missing output path")
			os.Exit(1)
		}
		tsOut := os.Args[2]
		must(bc.GenerateTsCode(tsOut))
	}
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

func (bc *BuseContext) NewPathDoc(name string) *Doc {
	return &Doc{
		bc:   bc,
		name: name,
	}
}

func (bc *BuseContext) NewBusegenRelativeDoc(relname string) *Doc {
	name := filepath.Join(bc.Dir, filepath.FromSlash(relname))
	return &Doc{
		bc:   bc,
		name: name,
	}
}

func must(err error) {
	if err != nil {
		if se, ok := err.(*errors.Error); ok {
			log.Fatal(se.ErrorStack())
		} else {
			log.Fatal(se.Error())
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
	dest := b.name
	log.Printf("Writing (%s)...", dest)
	must(os.MkdirAll(filepath.Dir(dest), 0755))
	must(ioutil.WriteFile(dest, bs, 0644))
}

func (bc *BuseContext) Timestamp() string {
	return time.Now().Format(time.Stamp)
}

func (bc *BuseContext) Revision() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = bc.Dir
	rev, err := cmd.CombinedOutput()
	must(err)
	return strings.TrimSpace(string(rev))
}
