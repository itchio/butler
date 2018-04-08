package elefant

import (
	"bufio"
	"bytes"
	"debug/elf"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/itchio/wharf/eos"
	"github.com/pkg/errors"
)

type TraceNode struct {
	Name     string
	FullPath string
	Info     *ElfInfo

	Children          []*TraceNode
	UnresolvedImports []string
}

type Cache struct {
	Nodes map[string]*TraceNode
}

func (c *Cache) add(tn *TraceNode) {
	c.Nodes[tn.FullPath] = tn
}

func Trace(info *ElfInfo, fullPath string) (*TraceNode, error) {
	fullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	root := &TraceNode{
		Name:     filepath.Base(fullPath),
		FullPath: fullPath,
		Info:     info,
	}

	searchPaths, err := getSearchPaths()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cache := &Cache{
		Nodes: make(map[string]*TraceNode),
	}
	cache.add(root)

	err = root.trace(cache, searchPaths)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return root, nil
}

func (n *TraceNode) trace(cache *Cache, searchPaths *SearchPaths) error {
	for _, imp := range n.Info.Imports {

		importPath := searchPaths.lookup(imp, n.Info.Arch)
		if importPath == "" {
			n.UnresolvedImports = append(n.UnresolvedImports, imp)
		} else {
			if cn, ok := cache.Nodes[importPath]; ok {
				// cool!
				n.Children = append(n.Children, cn)
			} else {
				err := func() error {
					f, err := eos.Open(importPath)
					if err != nil {
						return errors.WithStack(err)
					}
					defer f.Close()

					ei, err := Probe(f, nil)
					if err != nil {
						return errors.WithStack(err)
					}

					cn := &TraceNode{
						Name:     imp,
						FullPath: importPath,
						Info:     ei,
					}
					cache.add(cn)
					n.Children = append(n.Children, cn)

					return cn.trace(cache, searchPaths)
				}()
				if err != nil {
					return errors.WithStack(err)
				}
			}
		}
	}
	return nil
}

type stringifyContext struct {
	donePaths map[string]bool
}

func (n *TraceNode) String() string {
	return "\n" + n.stringify(&stringifyContext{
		donePaths: make(map[string]bool),
	})
}

func (n *TraceNode) stringify(ctx *stringifyContext) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("- %s", n.FullPath))

	for _, ui := range n.UnresolvedImports {
		lines = append(lines, fmt.Sprintf("  - MISSING %s", ui))
	}
	for _, c := range n.Children {
		if _, ok := ctx.donePaths[c.FullPath]; ok {
			continue
		}
		ctx.donePaths[c.FullPath] = true

		for _, l := range strings.Split(c.stringify(ctx), "\n") {
			lines = append(lines, fmt.Sprintf("  %s", l))
		}
	}
	return strings.Join(lines, "\n")
}

type SearchPaths struct {
	Paths []string

	archCache map[string]Arch
}

func (sp *SearchPaths) getArch(fullpath string) Arch {
	if sp.archCache == nil {
		sp.archCache = make(map[string]Arch)
	}

	if arch, ok := sp.archCache[fullpath]; ok {
		return arch
	}

	var arch = ArchUnknown
	ef, err := elf.Open(fullpath)
	if err == nil {
		defer ef.Close()
		switch ef.Machine {
		case elf.EM_386:
			arch = Arch386
		case elf.EM_X86_64:
			arch = ArchAmd64
		}
	}

	sp.archCache[fullpath] = arch
	return arch
}

func (sp *SearchPaths) lookup(name string, arch Arch) string {
	for _, dir := range sp.Paths {
		candidatePath := filepath.Join(dir, name)
		candidateArch := sp.getArch(candidatePath)
		if candidateArch == arch {
			return candidatePath
		}
	}
	return ""
}

func (sp *SearchPaths) addPath(path string) {
	sp.Paths = append(sp.Paths, path)
}

func getSearchPaths() (*SearchPaths, error) {
	sp := &SearchPaths{}
	sp.addPath("/usr/lib") // this one is standard

	err := sp.parseConfig("/etc/ld.so.conf")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return sp, nil
}

var ldSoConfCommentRe = regexp.MustCompile("#.*$")

// cf. https://www.daemon-systems.org/man/ld.so.conf.5.html
// we do not support hardware-dependent directives
func (sp *SearchPaths) parseConfig(configPath string) error {
	contents, err := ioutil.ReadFile(configPath)
	if err != nil {
		return errors.WithStack(err)
	}

	s := bufio.NewScanner(bytes.NewReader(contents))
	for s.Scan() {
		lineWithComments := s.Text()
		line := ldSoConfCommentRe.ReplaceAllLiteralString(lineWithComments, "")
		line = strings.TrimSpace(line)

		switch {
		case len(line) == 0:
			// ignore empty / comment-only lines
			continue
		case strings.HasPrefix(line, "include "):
			includePath := strings.TrimSpace(strings.TrimPrefix(line, "include "))

			files, err := filepath.Glob(includePath)
			if err != nil {
				return errors.WithStack(err)
			}

			for _, f := range files {
				err = sp.parseConfig(f)
				if err != nil {
					return errors.WithStack(err)
				}
			}
		case strings.HasPrefix(line, "/"):
			sp.addPath(line)
		}
	}

	return nil
}
