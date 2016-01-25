package main

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/itchio/wharf.proto/megafile"
	"github.com/itchio/wharf.proto/rsync"

	"gopkg.in/kothar/brotli-go.v0/dec"
)

const MP_BLOCK_SIZE = 16 * 1024 // 16k

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

func readRepoInfo(reader io.Reader) *megafile.RepoInfo {
	info := &megafile.RepoInfo{
		BlockSize: MP_BLOCK_SIZE,
	}
	expectMagic(reader, MP_REPO_INFO)

	expectMagic(reader, MP_NUM_BLOCKS)
	must(binary.Read(reader, binary.LittleEndian, &info.NumBlocks))

	var numDirs, numFiles, numSymlinks int32
	var dir megafile.Dir
	var file megafile.File
	var symlink megafile.Symlink

	expectMagic(reader, MP_DIRS)
	must(binary.Read(reader, binary.LittleEndian, &numDirs))
	for i := int32(0); i < numDirs; i++ {
		must(readString(reader, &dir.Path))
		must(binary.Read(reader, binary.LittleEndian, &dir.Mode))
		info.Dirs = append(info.Dirs, dir)
	}

	expectMagic(reader, MP_FILES)
	must(binary.Read(reader, binary.LittleEndian, &numFiles))
	for i := int32(0); i < numFiles; i++ {
		must(readString(reader, &file.Path))
		must(binary.Read(reader, binary.LittleEndian, &file.Mode))
		must(binary.Read(reader, binary.LittleEndian, &file.Size))
		must(binary.Read(reader, binary.LittleEndian, &file.BlockIndex))
		must(binary.Read(reader, binary.LittleEndian, &file.BlockIndexEnd))
		info.Files = append(info.Files, file)
	}

	expectMagic(reader, MP_SYMLINKS)
	must(binary.Read(reader, binary.LittleEndian, &numSymlinks))
	for i := int32(0); i < numSymlinks; i++ {
		must(readString(reader, &symlink.Path))
		must(binary.Read(reader, binary.LittleEndian, &symlink.Mode))
		must(readString(reader, &symlink.Dest))
		info.Symlinks = append(info.Symlinks, symlink)
	}

	return info
}

func megapatch(patch string, target string, output string) {
	compressedReader, err := os.Open(patch)
	must(err)

	patchReader := dec.NewBrotliReader(compressedReader)
	expectMagic(patchReader, MP_MAGIC)

	targetInfo := readRepoInfo(patchReader)
	printRepoStats(targetInfo, target)

	sourceInfo := readRepoInfo(patchReader)
	printRepoStats(sourceInfo, output)

	expectMagic(patchReader, MP_RSYNC_OPS)

	sourceWriter, err := sourceInfo.NewWriter(output)
	must(err)

	targetReader := targetInfo.NewReader(target)

	rs := &rsync.RSync{
		BlockSize: sourceInfo.BlockSize,
	}

	ops := make(chan rsync.Operation)

	go (func() {
		defer close(ops)
		numOps := 0

		var magic int32
		reading := true

		for reading {
			must(binary.Read(patchReader, binary.LittleEndian, &magic))

			switch magic {
			case MP_RSYNC_OP:
				numOps++
				var op rsync.Operation
				var typ byte
				must(binary.Read(patchReader, binary.LittleEndian, &typ))
				op.Type = rsync.OpType(typ)

				switch op.Type {
				case rsync.OpBlock:
					must(binary.Read(patchReader, binary.LittleEndian, &op.BlockIndex))
				case rsync.OpBlockRange:
					must(binary.Read(patchReader, binary.LittleEndian, &op.BlockIndex))
					must(binary.Read(patchReader, binary.LittleEndian, &op.BlockIndexEnd))
				case rsync.OpData:
					var buflen int64
					must(binary.Read(patchReader, binary.LittleEndian, &buflen))

					Logf("reading data of len %d", buflen)
					buf := make([]byte, buflen)
					_, err := io.ReadFull(patchReader, buf)
					must(err)
					op.Data = buf
				default:
					Dief("corrupted patch: unknown rsync op type %d", op.Type)
				}

				Logf("Applying op of type %d", op.Type)
				ops <- op

			case MP_EOF:
				// cool!
				Logf("Read %d ops", numOps)
				Logf("Cool, you did it :)")
				reading = false
			default:
				Dief("corrupted patch: unknown magic %d", magic)
			}
		}
	})()

	err = rs.ApplyDelta(sourceWriter, targetReader, ops)
	if err != nil {
		Dief("while applying delta: %s", err.Error())
	}

	Logf("Rebuilt source in %s", output)
	outputInfo, err := megafile.Walk(output, sourceInfo.BlockSize)
	must(err)
	printRepoStats(outputInfo, output)
}
