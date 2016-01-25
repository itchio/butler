package main

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/dustin/go-humanize"
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
		Dief("corrupted megarecipe (expected magic %#x)", expected)
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

func apply(recipe string, target string, output string) {
	compressedReader, err := os.Open(recipe)
	must(err)

	recipeReader := dec.NewBrotliReader(compressedReader)
	expectMagic(recipeReader, MP_MAGIC)

	targetInfo := readRepoInfo(recipeReader)
	printRepoStats(targetInfo, target)

	sourceInfo := readRepoInfo(recipeReader)
	printRepoStats(sourceInfo, output)

	expectMagic(recipeReader, MP_RSYNC_OPS)

	sourceWriter, err := sourceInfo.NewWriter(output)
	must(err)

	targetReader := targetInfo.NewReader(target)

	rs := &rsync.RSync{
		BlockSize: sourceInfo.BlockSize,
	}

	ops := make(chan rsync.Operation)

	go (func() {
		defer close(ops)
		totalOps := 0
		opsCount := []int{0, 0, 0}
		opsBytes := []int64{0, 0, 0}

		var magic int32
		reading := true

		for reading {
			must(binary.Read(recipeReader, binary.LittleEndian, &magic))

			switch magic {
			case MP_RSYNC_OP:
				totalOps++
				var op rsync.Operation
				var typ byte
				must(binary.Read(recipeReader, binary.LittleEndian, &typ))
				op.Type = rsync.OpType(typ)
				opsCount[op.Type]++

				switch op.Type {
				case rsync.OpBlock:
					must(binary.Read(recipeReader, binary.LittleEndian, &op.BlockIndex))
					opsBytes[op.Type] += int64(sourceInfo.BlockSize)
				case rsync.OpBlockRange:
					must(binary.Read(recipeReader, binary.LittleEndian, &op.BlockIndex))
					must(binary.Read(recipeReader, binary.LittleEndian, &op.BlockIndexEnd))
					opsBytes[op.Type] += int64(sourceInfo.BlockSize) * int64(op.BlockIndexEnd-op.BlockIndex)
				case rsync.OpData:
					var buflen int64
					must(binary.Read(recipeReader, binary.LittleEndian, &buflen))
					opsBytes[op.Type] += buflen

					buf := make([]byte, buflen)
					_, err := io.ReadFull(recipeReader, buf)
					must(err)
					op.Data = buf
				default:
					Dief("Corrupt recipe: unknown rsync op %d", op.Type)
				}
				ops <- op

			case MP_EOF:
				// cool!
				if *appArgs.verbose {
					Logf("recipe had %d ops:", totalOps)
					for i, name := range []string{"block", "block-range", "data"} {
						Logf("%10s %s (%d ops)", name, humanize.Bytes(uint64(opsBytes[i])), opsCount[i])
					}
				}
				Logf("Cool, you did it :)")
				reading = false
			default:
				Dief("Corrupt recipe: unknown magic %d (expected RSYNC_OP or EOF)", magic)
			}
		}
	})()

	err = rs.ApplyRecipe(sourceWriter, targetReader, ops)
	if err != nil {
		Dief("While applying delta: %s", err.Error())
	}

	if *appArgs.verbose {
		Logf("Rebuilt source in %s", output)
	}
	outputInfo, err := megafile.Walk(output, sourceInfo.BlockSize)
	must(err)
	printRepoStats(outputInfo, output)
}
