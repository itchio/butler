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
)

type GenerousContext struct {
	Dir string
}

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Printf("generous is a documentation & bindings generator for butlerd")
		log.Printf("")
		log.Printf("Usage: generous (godocs|ts [OUT])")
		log.Printf("  - godocs: generate directly in the $GOPATH tree")
		log.Printf("  - ts: give a target path to generate")
		os.Exit(1)
	}
	mode := os.Args[1]

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Fatalf("$GOPATH must be set")
	}

	baseDir := filepath.Join(gopath, "src", "github.com", "itchio", "butler", "butlerd", "generous")
	_, err := os.Stat(baseDir)
	must(err)
	log.Printf("Base dir: (%s)", baseDir)

	gc := &GenerousContext{
		Dir: baseDir,
	}

	switch mode {
	case "godocs":
		must(gc.GenerateDocs())
		must(gc.GenerateGoCode())
		must(gc.GenerateSpec())
	case "ts":
		if len(os.Args) < 2 {
			log.Printf("generous ts: missing output path")
			os.Exit(1)
		}
		tsOut := os.Args[2]
		must(gc.GenerateTsCode(tsOut))
	}
}

func (gc *GenerousContext) Task(task string) {
	log.Printf("")
	log.Printf("=========================")
	log.Printf(">> %s", task)
	log.Printf("=========================")
}

func (gc *GenerousContext) ReadFile(file string) string {
	bs, err := ioutil.ReadFile(filepath.Join(gc.Dir, file))
	must(err)
	return string(bs)
}

func (gc *GenerousContext) NewPathDoc(name string) *Doc {
	return &Doc{
		gc:   gc,
		name: name,
	}
}

func (gc *GenerousContext) NewGenerousRelativeDoc(relname string) *Doc {
	name := filepath.Join(gc.Dir, filepath.FromSlash(relname))
	return &Doc{
		gc:   gc,
		name: name,
	}
}

func must(err error) {
	if err != nil {
		log.Fatalf("%+v", err)
	}
}

//

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

func (gc *GenerousContext) Timestamp() string {
	return time.Now().Format(time.Stamp)
}

func (gc *GenerousContext) Revision() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = gc.Dir
	rev, err := cmd.CombinedOutput()
	must(err)
	return strings.TrimSpace(string(rev))
}
