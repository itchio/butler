package mitch

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/itchio/savior/seeksource"

	"github.com/itchio/arkive/zip"
	"github.com/itchio/wharf/pools/zippool"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/itchio/wharf/wsync"
)

func (s *Store) MakeUser(displayName string) *User {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	user := &User{
		Store:       s,
		ID:          s.serial(),
		Username:    s.slugify(displayName),
		DisplayName: displayName,
		Gamer:       true,
	}
	s.Users[user.ID] = user
	return user
}

func (u *User) MakeAPIKey() *APIKey {
	s := u.Store
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	apiKey := &APIKey{
		Store:  s,
		ID:     s.serial(),
		UserID: u.ID,
		Key:    fmt.Sprintf("%s-api-key", u.Username),
	}
	s.APIKeys[apiKey.ID] = apiKey
	return apiKey
}

func (u *User) MakeGame(title string) *Game {
	s := u.Store
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	game := &Game{
		Store:          s,
		ID:             s.serial(),
		Type:           "default",
		Classification: "game",
		UserID:         u.ID,
		Title:          title,
	}
	s.Games[game.ID] = game
	return game
}

func (g *Game) Publish() {
	g.Published = true
}

func (g *Game) MakeUpload(title string) *Upload {
	s := g.Store
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	upload := &Upload{
		Store:  s,
		ID:     s.serial(),
		GameID: g.ID,
		Type:   "default",
	}
	s.Uploads[upload.ID] = upload
	return upload
}

func (u *Upload) SetAllPlatforms() {
	u.PlatformWindows = true
	u.PlatformLinux = true
	u.PlatformMac = true
}

func (u *Upload) SetZipContents() {
	u.SetZipContentsCustom(func(ac *ArchiveContext) {
		ac.Entry("hello.txt").String("Just a test file")
	})
}

func (u *Upload) SetZipContentsCustom(f func(ac *ArchiveContext)) {
	ac := &ArchiveContext{
		Entries: make(map[string]*ArchiveEntry),
		Name:    fmt.Sprintf("upload-%d.zip", u.ID),
	}
	f(ac)
	u.SetHostedContents(ac.Name, ac.CompressZip())
}

func (u *Upload) PushBuild(f func(ac *ArchiveContext)) *Build {
	s := u.Store

	parentBuild := s.FindBuild(u.Head)
	b := &Build{
		Store: s,

		ID:       s.serial(),
		UploadID: u.ID,
		Version:  1,
	}
	if parentBuild != nil {
		b.ParentBuildID = parentBuild.ID
		b.Version = parentBuild.Version + 1
	}
	s.Builds[b.ID] = b

	ac := &ArchiveContext{
		Entries: make(map[string]*ArchiveEntry),
		Name:    fmt.Sprintf("build-%d.zip", b.ID),
	}
	f(ac)

	archiveFile := b.MakeFile("archive", "default")
	archiveFile.SetHostedContents(ac.Name, ac.CompressZip())

	archiveFile.Sign()
	if parentBuild != nil {
		archiveFile.Diff(parentBuild)
	}

	u.Storage = "build"
	u.Head = b.ID
	u.Filename = archiveFile.Filename
	u.Size = archiveFile.Size
	if u.ChannelName == "" {
		u.ChannelName = fmt.Sprintf("upload-%d", u.ID)
	}
	return b
}

func (b *Build) MakeFile(typ string, subtype string) *BuildFile {
	s := b.Store

	bf := &BuildFile{
		Store: s,

		ID:      s.serial(),
		BuildID: b.ID,
		Type:    typ,
		SubType: subtype,
	}
	s.BuildFiles[bf.ID] = bf
	return bf
}

func (b *Build) GetFile(typ string, subtype string) *BuildFile {
	s := b.Store

	return s.SelectBuildFile(NoSort(), Eq{
		"BuildID": b.ID,
		"Type":    typ,
		"SubType": subtype,
	})
}

func (u *Upload) SetHostedContents(filename string, contents []byte) {
	f := u.Store.UploadCDNFile(u.CDNPath(), filename, contents)
	u.Storage = "hosted"
	u.Filename = filename
	u.Size = f.Size
}

func (u *Upload) CDNPath() string {
	return fmt.Sprintf("/uploads/%d", u.ID)
}

func (bf *BuildFile) SetHostedContents(filename string, contents []byte) {
	f := bf.Store.UploadCDNFile(bf.CDNPath(), filename, contents)
	bf.Filename = filename
	bf.Size = f.Size
}

func (bf *BuildFile) Sign() *BuildFile {
	s := bf.Store
	if bf.Type != "archive" {
		panic("Can only sign 'archive' BuildFile")
	}

	b := s.FindBuild(bf.BuildID)
	if b == nil {
		panic("BuildFile without Build")
	}

	archiveCDNFile := s.CDNFiles[bf.CDNPath()]
	if archiveCDNFile == nil {
		panic("missing CDN File for archive BuildFile")
	}

	zr, err := zip.NewReader(bytes.NewReader(archiveCDNFile.Contents), archiveCDNFile.Size)
	must(err)

	container, err := tlc.WalkZip(zr, &tlc.WalkOpts{})
	must(err)

	pool := zippool.New(container, zr)
	sigBuf := new(bytes.Buffer)

	compression := compressionSettings()

	rawSigWire := wire.NewWriteContext(sigBuf)
	err = rawSigWire.WriteMagic(pwr.SignatureMagic)
	must(err)
	err = rawSigWire.WriteMessage(&pwr.SignatureHeader{
		Compression: &compression,
	})
	must(err)

	sigWire, err := pwr.CompressWire(rawSigWire, &compression)
	must(err)

	err = sigWire.WriteMessage(container)
	must(err)

	ctx := context.Background()
	consumer := &state.Consumer{}
	err = pwr.ComputeSignatureToWriter(ctx, container, pool, consumer, func(hash wsync.BlockHash) error {
		return sigWire.WriteMessage(&pwr.BlockHash{
			WeakHash:   hash.WeakHash,
			StrongHash: hash.StrongHash,
		})
	})
	must(err)

	err = sigWire.Close()
	must(err)

	sf := b.MakeFile("signature", "default")
	filename := fmt.Sprintf("build-%d-signature", b.ID)
	sf.SetHostedContents(filename, sigBuf.Bytes())
	return sf
}

func (bf *BuildFile) Diff(parentBuild *Build) *BuildFile {
	s := bf.Store
	if bf.Type != "archive" {
		panic("Can only diff 'archive' BuildFile")
	}

	b := s.FindBuild(bf.BuildID)
	if b == nil {
		panic("BuildFile without Build")
	}

	archiveCDNFile := s.CDNFiles[bf.CDNPath()]
	if archiveCDNFile == nil {
		panic("missing CDN File for archive BuildFile")
	}

	sourceZr, err := zip.NewReader(bytes.NewReader(archiveCDNFile.Contents), archiveCDNFile.Size)
	must(err)

	sourceContainer, err := tlc.WalkZip(sourceZr, &tlc.WalkOpts{})
	must(err)

	parentSig := parentBuild.GetFile("signature", "default")
	if parentSig == nil {
		panic("parent build is missing a signature")
	}

	parentSigCDNFile := s.CDNFiles[parentSig.CDNPath()]
	if parentSigCDNFile == nil {
		panic("missing CDN file for parent signature")
	}

	sigReader := seeksource.FromBytes(parentSigCDNFile.Contents)
	_, err = sigReader.Resume(nil)
	must(err)

	ctx := context.Background()

	sigInfo, err := pwr.ReadSignature(ctx, sigReader)
	must(err)

	sourcePool := zippool.New(sourceContainer, sourceZr)

	compression := compressionSettings()
	dctx := pwr.DiffContext{
		Compression: &compression,

		SourceContainer: sourceContainer,
		Pool:            sourcePool,

		TargetContainer: sigInfo.Container,
		TargetSignature: sigInfo.Hashes,
	}
	patchBuf := new(bytes.Buffer)
	err = dctx.WritePatch(ctx, patchBuf, ioutil.Discard)
	must(err)

	patchFile := b.MakeFile("patch", "default")
	filename := fmt.Sprintf("patch-%d.pwr", b.ID)
	patchFile.SetHostedContents(filename, patchBuf.Bytes())

	return patchFile
}

func compressionSettings() pwr.CompressionSettings {
	return pwr.CompressionSettings{
		Algorithm: pwr.CompressionAlgorithm_NONE,
	}
}

func (b *BuildFile) CDNPath() string {
	return fmt.Sprintf("/build-files/%d", b.ID)
}

func (s *Store) UploadCDNFile(path string, filename string, contents []byte) *CDNFile {
	f := &CDNFile{
		Path:     path,
		Filename: filename,
		Size:     int64(len(contents)),
		Contents: contents,
	}
	s.CDNFiles[path] = f
	return f
}
