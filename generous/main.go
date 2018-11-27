package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type GenerousContext struct {
	Config     Config
	ServiceDir string
	RootDir    string
}

var (
	app        = kingpin.New("generous", "A documentation & bindings generator for butlerd")
	godocsCmd  = app.Command("godocs", "Generate Markdown documentation, JSON spec & Go scaffolding")
	tsCmd      = app.Command("ts", "Generate TypeScript bindings")
	serviceDir = ""
	tsOut      = ""
)

func main() {
	godocsCmd.Arg("dir", "Service directory").Required().ExistingDirVar(&serviceDir)

	tsCmd.Arg("dir", "Service directory").Required().ExistingDirVar(&serviceDir)
	tsCmd.Arg("out", "Output directory").Required().StringVar(&tsOut)

	log.SetFlags(0)
	fullCmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Fatalf("$GOPATH must be set")
	}

	serviceDir, err := filepath.Abs(serviceDir)
	must(err)
	_, err = os.Stat(serviceDir)
	must(err)
	log.Printf("Service dir: (%s)", serviceDir)

	configPath := filepath.Join(serviceDir, "generous-config.json")
	configBytes, err := ioutil.ReadFile(configPath)
	must(err)
	var config Config
	must(json.Unmarshal(configBytes, &config))
	log.Printf("Package: (%s)", config.Input.Package)

	rootDir := serviceDir
	foundRootDir := false
	for levels := 4; levels >= 0; levels-- {
		vendorManifest := filepath.Join(rootDir, "vendor", "manifest")
		_, err := os.Stat(vendorManifest)
		if err == nil {
			foundRootDir = true
			break
		}
		rootDir = filepath.Dir(rootDir)
	}

	if !foundRootDir {
		must(errors.Errorf("Could not find root repository for service dir %s", serviceDir))
	}
	log.Printf("Root dir: (%s)", rootDir)

	gc := &GenerousContext{
		Config:     config,
		ServiceDir: serviceDir,
	}

	switch fullCmd {
	case godocsCmd.FullCommand():
		must(gc.GenerateDocs())
		must(gc.GenerateGoCode())
		must(gc.GenerateSpec())
	case tsCmd.FullCommand():
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
	bs, err := ioutil.ReadFile(filepath.Join(gc.ServiceDir, file))
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
	name := filepath.Join(gc.ServiceDir, filepath.FromSlash(relname))
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

func (gc *GenerousContext) Timestamp() string {
	return time.Now().Format(time.Stamp)
}

func (gc *GenerousContext) Revision() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = gc.ServiceDir
	rev, err := cmd.CombinedOutput()
	must(err)
	return strings.TrimSpace(string(rev))
}
