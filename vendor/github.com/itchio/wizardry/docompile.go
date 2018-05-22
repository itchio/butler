package main

import (
	"fmt"

	"github.com/itchio/wizardry/wizardry/wizcompiler"
	"github.com/itchio/wizardry/wizardry/wizparser"
	"github.com/pkg/errors"
)

func doCompile() error {
	magdir := *compileArgs.magdir

	NoLogf := func(format string, args ...interface{}) {}

	Logf := func(format string, args ...interface{}) {
		fmt.Println(fmt.Sprintf(format, args...))
	}

	pctx := &wizparser.ParseContext{
		Logf: NoLogf,
	}

	if *appArgs.debugParser {
		pctx.Logf = Logf
	}

	book := make(wizparser.Spellbook)
	err := pctx.ParseAll(magdir, book)
	if err != nil {
		return errors.WithStack(err)
	}

	err = wizcompiler.Compile(book, *compileArgs.output, *compileArgs.chatty, *compileArgs.emitComments, *compileArgs.pkg)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
