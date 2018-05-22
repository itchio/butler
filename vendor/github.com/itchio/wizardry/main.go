package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/itchio/butler/comm"
)

var (
	app = kingpin.New("wizardry", "A magic parser/interpreter/compiler")

	compileCmd  = app.Command("compile", "Compile a set of magic files into one .go file")
	identifyCmd = app.Command("identify", "Use a magic file to identify a target file")
)

var appArgs = struct {
	debugParser      *bool
	debugInterpreter *bool
}{
	app.Flag("debug-parser", "Turn on verbose parser output").Bool(),
	app.Flag("debug-interpreter", "Turn on verbose interpreter output").Bool(),
}

var identifyArgs = struct {
	magdir *string
	target *string
}{
	identifyCmd.Arg("magdir", "the folder of magic files to compile").Required().String(),
	identifyCmd.Arg("target", "path of the the file to identify").Required().String(),
}

var compileArgs = struct {
	magdir       *string
	output       *string
	chatty       *bool
	emitComments *bool
	pkg          *string
}{
	compileCmd.Arg("magdir", "the folder of magic files to compile").Required().String(),
	compileCmd.Flag("output", "the go file to generate").Short('o').Required().String(),
	compileCmd.Flag("chatty", "generate prints on every rule match").Bool(),
	compileCmd.Flag("emit-comments", "generate comments in the code").Bool(),
	compileCmd.Flag("package", "go package to generate").Default("main").String(),
}

func main() {
	app.HelpFlag.Short('h')
	app.Author("Amos Wenger <amos@itch.io>")

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		ctx, _ := app.ParseContext(os.Args[1:])
		app.FatalUsageContext(ctx, "%s\n", err.Error())
	}

	switch kingpin.MustParse(cmd, err) {
	case compileCmd.FullCommand():
		must(doCompile())
	case identifyCmd.FullCommand():
		must(doIdentify())
	}
}

func must(err error) {
	if err != nil {
		comm.Dief("%+v", err)
	}
}
