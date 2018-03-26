package tlc

import (
	"fmt"
	"sort"
)

func (c1 *Container) EnsureEqual(c2 *Container) error {
	dirs1 := sortedDirs(c1)
	dirs2 := sortedDirs(c2)

	if len(dirs1) != len(dirs2) {
		return fmt.Errorf("expected %d dirs, got %d dirs", len(dirs1), len(dirs2))
	}

	for i := range dirs1 {
		path1 := dirs1[i]
		path2 := dirs2[i]
		if path1 != path2 {
			return fmt.Errorf("expected dir %d to be %s, was %s", i, path1, path2)
		}
	}

	links1, linksmap1 := sortedLinks(c1)
	links2, linksmap2 := sortedLinks(c2)

	if len(links1) != len(links2) {
		return fmt.Errorf("expected %d symlinks, got %d symlinks", len(links1), len(links2))
	}

	for i := range links1 {
		path1 := links1[i]
		path2 := links2[i]
		if path1 != path2 {
			return fmt.Errorf("expected symlink %d to be %s, was %s", i, path1, path2)
		}

		dest1 := linksmap1[path1]
		dest2 := linksmap2[path2]
		if dest1 != dest2 {
			return fmt.Errorf("expected symlink %s to point to %s, pointed to %s", path1, dest1, dest2)
		}
	}

	files1, filesmap1 := sortedFiles(c1)
	files2, filesmap2 := sortedFiles(c2)

	if len(files1) != len(files2) {
		return fmt.Errorf("expected %d files, got %d files", len(files1), len(files2))
	}

	for i := range files1 {
		path1 := files1[i]
		path2 := files2[i]
		if path1 != path2 {
			return fmt.Errorf("expected file %d to be %s, was %s", i, path1, path2)
		}

		f1 := filesmap1[path1]
		f2 := filesmap2[path2]
		if f1.Size != f2.Size {
			return fmt.Errorf("expected file %s to have size %d, had size %d", path1, f1.Size, f2.Size)
		}
	}

	return nil
}

func sortedDirs(c *Container) []string {
	dirs := []string{}
	for _, d := range c.Dirs {
		dirs = append(dirs, d.Path)
	}
	sort.Strings(dirs)
	return dirs
}

func sortedLinks(c *Container) ([]string, map[string]string) {
	links := []string{}
	linksmap := make(map[string]string)
	for _, s := range c.Symlinks {
		links = append(links, s.Path)
		linksmap[s.Path] = s.Dest
	}
	sort.Strings(links)
	return links, linksmap
}

func sortedFiles(c *Container) ([]string, map[string]*File) {
	files := []string{}
	filesmap := make(map[string]*File)
	for _, f := range c.Files {
		files = append(files, f.Path)
		filesmap[f.Path] = f
	}
	sort.Strings(files)
	return files, filesmap
}
