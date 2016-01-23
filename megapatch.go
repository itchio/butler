package main

const (
	MP_MAGIC      = uint64(0xFEF5F04A)
	MP_NUM_BLOCKS = iota
	MP_FILES
	MP_DIRS
	MP_SYMLINKS
)

func megapatch(patch string, source string, output string) {
	Die("megapatch: stub!")
}
