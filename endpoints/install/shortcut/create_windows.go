//go:build windows
// +build windows

package shortcut

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/itchio/ox/winox"
	"github.com/scjalliance/comshim"
)

func Create(params CreateParams) error {
	err := validation.ValidateStruct(&params,
		validation.Field(&params.DisplayName, validation.Required),
		validation.Field(&params.URL, validation.Required),
		validation.Field(&params.Consumer, validation.Required),
	)
	if err != nil {
		return err
	}

	consumer := params.Consumer

	startMenuPath, err := winox.GetFolderPath(winox.FolderTypePrograms)
	if err != nil {
		return err
	}

	itchCorpPath := filepath.Join(startMenuPath, "Itch Corp")
	err = os.MkdirAll(itchCorpPath, 0o755)
	if err != nil {
		return err
	}

	shortcutName := fmt.Sprintf("%s.url", sanitizeFileName(params.DisplayName))
	shortcutPath := filepath.Join(itchCorpPath, shortcutName)

	comshim.Add(1)
	defer comshim.Done()

	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer oleShellObject.Release()

	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer wshell.Release()

	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", shortcutPath)
	if err != nil {
		return err
	}
	idispatch := cs.ToIDispatch()
	oleutil.PutProperty(idispatch, "TargetPath", params.URL)
	_, err = oleutil.CallMethod(idispatch, "Save")
	if err != nil {
		return err
	}
	consumer.Infof("Created shortcut (%s)", shortcutPath)

	return nil
}

var anyAmountOfSpaces = regexp.MustCompile(`\s+`)

func sanitizeFileName(s string) string {
	var forbidden = []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, f := range forbidden {
		s = strings.ReplaceAll(s, f, "")
	}

	var reserved = []string{"con", "prn", "aux", "nul", "com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8",
		"com9", "lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7", "lpt8", "lpt9"}

	lowerS := strings.ToLower(s)
	for _, r := range reserved {
		if lowerS == r {
			s = fmt.Sprintf("%s_", s)
			lowerS = strings.ToLower(s)
		}
	}

	s = anyAmountOfSpaces.ReplaceAllString(s, " ")
	return s
}
