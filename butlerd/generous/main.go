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

// generousContext regroups global state for the generous tool
// along with a few utility methods
type generousContext struct {
	Dir string
}

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Printf("generous is a documentation & bindings generator for butlerd")
		log.Printf("")
		log.Printf("Usage: generous (godocs|ts [OUT])")
		log.Printf("  - godocs: generate directly in the butler sources")
		log.Printf("  - ts: give a target path to generate")
		os.Exit(1)
	}
	mode := os.Args[1]

	baseDir := getGoPackageDir("github.com/itchio/butler/butlerd/generous")
	log.Printf("Base dir: (%s)", baseDir)

	_, err := os.Stat(baseDir)
	must(err)

	gc := &generousContext{
		Dir: baseDir,
	}

	switch mode {
	case "godocs":
		must(gc.generateDocs())
		must(gc.generateGoCode())
		must(gc.generateSpec())
	case "ts":
		if len(os.Args) < 2 {
			log.Printf("generous ts: missing output path")
			os.Exit(1)
		}
		tsOut := os.Args[2]
		must(gc.generateTsCode(tsOut))
	}
}

func getGoPackageDir(pkg string) string {
	bs, err := exec.Command("go", "list", "-f", "{{ .Dir }}", pkg).Output()
	must(err)
	return strings.TrimSpace(string(bs))
}

func (gc *generousContext) task(task string) {
	log.Printf("")
	log.Printf("=========================")
	log.Printf(">> %s", task)
	log.Printf("=========================")
}

func (gc *generousContext) readFile(file string) string {
	bs, err := ioutil.ReadFile(filepath.Join(gc.Dir, file))
	must(err)
	return string(bs)
}

func (gc *generousContext) newPathDoc(name string) *document {
	return &document{
		gc:   gc,
		name: name,
	}
}

func (gc *generousContext) newGenerousRelativeDoc(relname string) *document {
	name := filepath.Join(gc.Dir, filepath.FromSlash(relname))
	return &document{
		gc:   gc,
		name: name,
	}
}

func must(err error) {
	if err != nil {
		log.Fatalf("%+v", err)
	}
}

type document struct {
	name string
	gc   *generousContext

	doc string
	buf string
}

func (d *document) load(doc string) {
	d.doc = doc
}

func (d *document) line(msg string, args ...interface{}) {
	d.buf += fmt.Sprintf(msg, args...)
	d.buf += "\n"
}

func (d *document) commit(name string) {
	if name == "" {
		d.doc = d.buf
	} else {
		d.doc = strings.Replace(d.doc, name, d.buf, 1)
	}
	d.buf = ""
}

func (d *document) write() {
	bs := []byte(d.doc)
	dest := d.name
	log.Printf("Writing (%s)...", dest)
	must(os.MkdirAll(filepath.Dir(dest), 0755))
	must(ioutil.WriteFile(dest, bs, 0644))
}

func (gc *generousContext) timestamp() string {
	return time.Now().Format(time.Stamp)
}

func (gc *generousContext) revision() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = gc.Dir
	rev, err := cmd.CombinedOutput()
	must(err)
	return strings.TrimSpace(string(rev))
}
