package main

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/itchio/wharf.proto/megafile"

	"gopkg.in/kothar/brotli-go.v0/dec"
)

const (
	MP_MAGIC = int32(iota + 0xFEF5F00)
	MP_REPO_INFO
	MP_NUM_BLOCKS
	MP_FILES
	MP_DIRS
	MP_SYMLINKS
	MP_RSYNC_OPS
	MP_RSYNC_OP
	MP_EOF
)

func expectMagic(reader io.Reader, expected int32) {
	var magic int32
	must(binary.Read(reader, binary.LittleEndian, &magic))
	if magic != expected {
		Dief("corrupted megapatch (expected magic %#x)", expected)
	}
}

func readString(r io.Reader, s *string) error {
	var slen int32
	err := binary.Read(r, binary.LittleEndian, &slen)
	if err != nil {
		return err
	}

	var buf = make([]byte, slen)
	_, err = r.Read(buf)
	if err != nil {
		return err
	}

	*s = string(buf)
	return nil
}

func readRepoInfo(reader io.Reader, info *megafile.RepoInfo) {
	Log("reading repo info")
	expectMagic(reader, MP_REPO_INFO)

	Log("reading num blocks")
	expectMagic(reader, MP_NUM_BLOCKS)
	must(binary.Read(reader, binary.LittleEndian, &info.NumBlocks))

	var numDirs, numFiles, numSymlinks int32
	var dir megafile.Dir
	var file megafile.File
	var symlink megafile.Symlink

	Log("reading dirs")
	expectMagic(reader, MP_DIRS)
	must(binary.Read(reader, binary.LittleEndian, &numDirs))
	for i := int32(0); i < numDirs; i++ {
		must(readString(reader, &dir.Path))
		must(binary.Read(reader, binary.LittleEndian, &dir.Mode))
	}

	Log("reading files")
	expectMagic(reader, MP_FILES)
	must(binary.Read(reader, binary.LittleEndian, &numFiles))
	for i := int32(0); i < numFiles; i++ {
		must(readString(reader, &file.Path))
		must(binary.Read(reader, binary.LittleEndian, &file.Mode))
		must(binary.Read(reader, binary.LittleEndian, &file.Size))
		must(binary.Read(reader, binary.LittleEndian, &file.BlockIndex))
		must(binary.Read(reader, binary.LittleEndian, &file.BlockIndexEnd))
	}

	Log("reading symlinks")
	expectMagic(reader, MP_SYMLINKS)
	must(binary.Read(reader, binary.LittleEndian, &numSymlinks))
	for i := int32(0); i < numSymlinks; i++ {
		must(readString(reader, &symlink.Path))
		must(binary.Read(reader, binary.LittleEndian, &symlink.Mode))
		must(binary.Read(reader, binary.LittleEndian, &symlink.Dest))
	}
}

func megapatch(patch string, source string, output string) {
	compressedReader, err := os.Open(patch)
	must(err)

	patchReader := dec.NewBrotliReader(compressedReader)
	expectMagic(patchReader, MP_MAGIC)

	targetInfo := &megafile.RepoInfo{}
	readRepoInfo(patchReader, targetInfo)

	sourceInfo := &megafile.RepoInfo{}
	readRepoInfo(patchReader, sourceInfo)

	expectMagic(patchReader, MP_RSYNC_OPS)

	Die("megapatch: stub!")
}
