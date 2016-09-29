package main

import "fmt"

func heal(dir string, wounds string, source string) {
	must(doHeal(dir, wounds, source))
}

func doHeal(dir string, wounds string, source string) error {
	return fmt.Errorf("heal: stub")
}
