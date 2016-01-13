package main

import "os"

func mkdir(dir string) {
	err := os.MkdirAll(dir, DIR_MODE)
	if err != nil {
		Dief(err.Error())
	}
	Logf("[success] mkdir -p %s", dir)
}

func wipe(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		Dief(err.Error())
	}
	Logf("[success] rm -rf %s", path)
}

func ditto(src string, dst string) {
	Dief("ditto: stub")
}
