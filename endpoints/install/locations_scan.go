package install

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/ox"

	"github.com/google/uuid"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/pkg/errors"
)

type scanContext struct {
	rc               *butlerd.RequestContext
	conn             *sqlite.Conn
	legacyMarketPath string
	newByID          map[string]*importedCave

	existingCaves []existingCave
	existingByID  map[string]*models.Cave

	installLocations []*models.InstallLocation

	numScanned int64
	tasks      []*task
}

type importedCave struct {
	cave    *models.Cave
	receipt *bfs.Receipt
}

type existingCave struct {
	ID                string
	InstallLocationID string
	InstallFolderName string
}

func InstallLocationsScan(rc *butlerd.RequestContext, params butlerd.InstallLocationsScanParams) (*butlerd.InstallLocationsScanResult, error) {
	consumer := rc.Consumer
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	sc := &scanContext{
		rc:               rc,
		conn:             conn,
		legacyMarketPath: params.LegacyMarketPath,
		newByID:          make(map[string]*importedCave),
		existingByID:     make(map[string]*models.Cave),
	}
	err := sc.Do()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	numFound := int64(len(sc.newByID))
	var numSaved int64

	if numFound > 0 {
		confirmRes, err := messages.InstallLocationsScanConfirmImport.Call(rc, butlerd.InstallLocationsScanConfirmImportParams{
			NumItems: numFound,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if confirmRes.Confirm {
			for _, ic := range sc.newByID {
				err := models.HadesContext().Save(conn, ic.cave,
					hades.Assoc("Game"),
					hades.Assoc("Upload"),
					hades.Assoc("Build"),
				)
				if err != nil {
					consumer.Errorf("Could not import: %s", err.Error())
				} else {
					numSaved++
				}

				InstallFolder := sc.getInstallLocation(ic.cave.InstallLocationID).GetInstallFolder(ic.cave.InstallFolderName)
				err = ic.receipt.WriteReceipt(InstallFolder)
				if err != nil {
					consumer.Errorf("Could not write receipt: %s", err.Error())
				}
			}
		} else {
			consumer.Infof("Not importing anything by user's request")
		}
	} else {
		consumer.Infof("No items found")
	}

	res := &butlerd.InstallLocationsScanResult{
		NumFoundItems:    numFound,
		NumImportedItems: numSaved,
	}
	return res, nil
}

func (sc *scanContext) Do() error {
	startTime := time.Now()

	rc := sc.rc
	consumer := rc.Consumer

	models.MustSelect(sc.conn, &sc.installLocations, builder.NewCond(), hades.Search{})

	hc := models.HadesContext()
	models.MustExec(sc.conn,
		builder.Select("id", "install_location_id", "install_folder_name").From("caves"),
		hc.IntoRowsScanner(&sc.existingCaves),
	)

	if sc.legacyMarketPath != "" {
		err := sc.DoMarket()
		if err != nil {
			return errors.WithStack(err)
		}
	}

	err := sc.DoInstallLocations()
	if err != nil {
		return errors.WithStack(err)
	}

	if len(sc.tasks) == 0 {
		return nil
	}

	consumer.Opf("Scanning %d entries", len(sc.tasks))

	rc.StartProgress()
	for i, t := range sc.tasks {
		consumer.Progress(float64(i) / float64(len(sc.tasks)))
		err := sc.importLegacyCave(t.legacyCave, t.files)
		if err != nil {
			consumer.Warnf(err.Error())
		}
	}
	rc.EndProgress()

	consumer.Statf("Scanned %d entries in %s", sc.numScanned, time.Since(startTime))
	consumer.Statf("Found %d caves to import", len(sc.newByID))
	for _, ic := range sc.newByID {
		c := ic.cave
		consumer.Infof("- %s", operate.GameToString(c.Game))
		operate.LogUpload(consumer, c.Upload, c.Build)
		consumer.Infof("  %s @ %s",
			progress.FormatBytes(c.InstalledSize),
			sc.getInstallLocation(c.InstallLocationID).GetInstallFolder(c.InstallFolderName),
		)
		consumer.Infof("")
	}

	return nil
}

func (sc *scanContext) DoMarket() error {
	rc := sc.rc
	consumer := rc.Consumer

	cavesPath := filepath.Join(sc.legacyMarketPath, "caves")
	entries, err := ioutil.ReadDir(cavesPath)
	if err != nil {
		return nil
	}

	markerPath := filepath.Join(sc.legacyMarketPath, ".scanned")
	{
		_, err := ioutil.ReadFile(markerPath)
		if err == nil {
			// already scanned, ignore
			return nil
		}

		msg := fmt.Sprintf("scanned %s", time.Now().Format(time.RFC3339Nano))
		err = ioutil.WriteFile(markerPath, []byte(msg), 0644)
		if err != nil {
			consumer.Warnf("Could not write marker: %+v", err)
		}
	}

	handleEntryPanics := func(entry os.FileInfo) error {
		legCaveBytes, err := ioutil.ReadFile(filepath.Join(cavesPath, entry.Name()))
		if err != nil {
			return errors.WithStack(err)
		}

		legCave := &legacyCave{}
		err = json.Unmarshal(legCaveBytes, legCave)
		if err != nil {
			return errors.WithStack(err)
		}

		if sc.hasCave(legCave.ID) {
			return nil
		}

		sc.queue(&task{legCave, nil})
		return nil
	}

	handleEntry := func(entry os.FileInfo) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if rErr, ok := r.(error); ok {
					err = errors.WithStack(rErr)
				} else {
					err = errors.Errorf("%v", r)
				}
			}
		}()
		err = handleEntryPanics(entry)
		return
	}

	for _, entry := range entries {
		err := handleEntry(entry)
		if err != nil {
			consumer.Errorf("While handling entry %s: %s", entry.Name(), err.Error())
		}
	}
	return nil
}

func (sc *scanContext) DoInstallLocations() error {
	consumer := sc.rc.Consumer
	for _, il := range sc.installLocations {
		consumer.Opf("Scanning install location %s...", il.Path)
		err := sc.DoInstallLocation(il)
		if err != nil {
			consumer.Warnf("Could not process install location %s: %+v", il.Path, err)
			consumer.Infof("Skipping...")
		}
	}

	return nil
}

func (sc *scanContext) DoInstallLocation(il *models.InstallLocation) error {
	rc := sc.rc
	consumer := rc.Consumer

	entries, err := ioutil.ReadDir(il.Path)
	if err != nil {
		return errors.WithStack(err)
	}

	handleEntryPanics := func(entry os.FileInfo) error {
		InstallFolderName := entry.Name()
		InstallFolder := filepath.Join(il.Path, InstallFolderName)

		if InstallFolderName == "downloads" {
			// definitely not a cave folder, skip
			return nil
		}

		if sc.hasCaveAtLoc(il.ID, InstallFolderName) {
			// already have cave, skip
			return nil
		}

		dotItchPath := filepath.Join(InstallFolder, ".itch")
		_, err := os.Stat(dotItchPath)
		if err != nil {
			// no .itch folder, skip
			return nil
		}

		receipt, err := bfs.ReadReceipt(InstallFolder)
		if err != nil {
			consumer.Warnf("While reading receipt in (%s): %s", InstallFolder, err.Error())
		}

		legacyReceiptPath := filepath.Join(dotItchPath, "receipt.json")
		legacyReceiptBytes, _ := ioutil.ReadFile(legacyReceiptPath)

		if receipt != nil {
			var buildID int64
			if receipt.Build != nil {
				buildID = receipt.Build.ID
			}

			receiptStats, err := os.Stat(bfs.ReceiptPath(InstallFolder))
			if err != nil {
				return errors.WithStack(err)
			}

			legCave := &legacyCave{
				ID:       uuid.New().String(),
				GameID:   receipt.Game.ID,
				UploadID: receipt.Upload.ID,
				BuildID:  buildID,

				// no play stats unfortunately
				LastTouched: 0,
				SecondsRun:  0,

				InstalledAt: receiptStats.ModTime().UTC().Format(time.RFC3339),

				PathScheme:      2,
				InstallLocation: il.ID,
				InstallFolder:   InstallFolderName,
			}
			sc.queue(&task{legCave, receipt.Files})
		} else {
			consumer.Infof("Reading legacy itch recept from %s", legacyReceiptPath)

			lr := &legacyReceipt{}
			err := json.Unmarshal(legacyReceiptBytes, lr)
			if err != nil {
				consumer.Errorf("Could not unmarshal legacy itch receipt: %s", err.Error())
				consumer.Infof("Skipping...")
				return nil
			}

			if lr.Cave != nil && sc.hasCave(lr.Cave.ID) {
				return nil
			}

			if lr.Cave != nil {
				if lr.Cave.InstallLocation != il.ID {
					consumer.Infof("Mapping found location ID from (%s) to existing (%s)", lr.Cave.InstallLocation, il.ID)
					lr.Cave.InstallLocation = il.ID
				}
			}
			sc.queue(&task{lr.Cave, lr.Files})
		}

		return nil
	}

	handleEntry := func(entry os.FileInfo) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if rErr, ok := r.(error); ok {
					err = errors.WithStack(rErr)
				} else {
					err = errors.Errorf("%v", r)
				}
			}
		}()
		err = handleEntryPanics(entry)
		return
	}

	for _, entry := range entries {
		err := handleEntry(entry)
		if err != nil {
			consumer.Errorf("While handling entry %s: %s", entry.Name(), err.Error())
		}
	}
	return nil
}

func (sc *scanContext) importLegacyCave(legacyCave *legacyCave, files []string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if rErr, ok := r.(error); ok {
				err = errors.WithStack(rErr)
			} else {
				err = errors.Errorf("%v", r)
			}
		}
	}()
	err = sc.importLegacyCavePanics(legacyCave, files)
	return
}

func (sc *scanContext) importLegacyCavePanics(legacyCave *legacyCave, files []string) error {
	sc.numScanned++
	rc := sc.rc
	consumer := rc.Consumer

	if legacyCave == nil {
		consumer.Errorf("Nil cave, skipping...")
		return nil
	}
	if legacyCave.ID == "" {
		consumer.Errorf("No cave ID, skipping...")
		return nil
	}
	if sc.hasCave(legacyCave.ID) {
		// skip
		return nil
	}
	if legacyCave.UploadID == 0 {
		consumer.Errorf("No upload ID, skipping...")
		return nil
	}
	if legacyCave.InstallLocation == "" {
		consumer.Errorf("No install location, skipping...")
		return nil
	}
	if legacyCave.InstallFolder == "" {
		consumer.Errorf("No folder, skipping...")
		return nil
	}
	if legacyCave.PathScheme != 2 {
		consumer.Errorf("Unsupported path scheme, skipping...")
		return nil
	}
	if sc.hasCaveAtLoc(legacyCave.InstallLocation, legacyCave.InstallFolder) {
		// skip
		return nil
	}

	il := sc.getInstallLocation(legacyCave.InstallLocation)
	if il == nil {
		consumer.Errorf("Cave refers to non-existent install location (%s), skipping", legacyCave.InstallLocation)
		return nil
	}

	InstallFolderName := legacyCave.InstallFolder
	InstallFolder := il.GetInstallFolder(InstallFolderName)
	if _, err := os.Stat(InstallFolder); err != nil {
		consumer.Errorf("Skipping: %s", err.Error())
		return nil
	}

	consumer.Infof("Item (%s)", InstallFolder)

	legacyReceiptPath := filepath.Join(InstallFolder, ".itch", "receipt.json")

	access := operate.AccessForGameID(sc.conn, legacyCave.GameID)
	client := rc.Client(access.APIKey)

	gameRes, err := client.GetGame(itchio.GetGameParams{
		GameID:      legacyCave.GameID,
		Credentials: access.Credentials,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	game := gameRes.Game

	uploadsRes, err := client.ListGameUploads(itchio.ListGameUploadsParams{
		GameID:      game.ID,
		Credentials: access.Credentials,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	var upload *itchio.Upload
	for _, u := range uploadsRes.Uploads {
		if u.ID == legacyCave.UploadID {
			upload = u
			break
		}
	}

	if upload == nil {
		consumer.Errorf("Could not find upload %d on server, skipping...", legacyCave.UploadID)
		return nil
	}

	var build *itchio.Build
	if legacyCave.BuildID != 0 {
		buildsRes, err := client.GetBuild(itchio.GetBuildParams{
			BuildID:     legacyCave.BuildID,
			Credentials: access.Credentials,
		})
		if err != nil {
			consumer.Warnf("Could not find build %d: %v", legacyCave.BuildID, err)
			// TODO: do we actually queue it?
			consumer.Infof("...an update will automatically be queued")
		} else {
			build = buildsRes.Build
		}
	}

	var LastTouchedAt *time.Time
	if legacyCave.LastTouched != 0 {
		LastTouchedAt = fromJSTimestamp(legacyCave.LastTouched)
	}

	var InstalledAt *time.Time
	switch installedAt := legacyCave.InstalledAt.(type) {
	case float64:
		// as of itch v18.3.0
		// JSON numbers unmarshal to float by default
		consumer.Infof("Reading installed at from timestamp %v", installedAt)
		InstalledAt = fromJSTimestamp(installedAt)
	case string:
		// as of itch v18.0.0
		consumer.Infof("Reading installed at from date %v", installedAt)
		InstalledAt = fromJSDate(installedAt)
	default:
		// before itch v18.0.0
		consumer.Warnf("Unable to get installed at timestamp, going with legacy receipt mtime")
		legacyReceiptStats, err := os.Stat(legacyReceiptPath)
		if err == nil {
			receiptModTime := legacyReceiptStats.ModTime().UTC()
			InstalledAt = &receiptModTime
		}
	}

	cave := &models.Cave{
		ID:                legacyCave.ID,
		InstallLocationID: il.ID,
		InstallFolderName: InstallFolderName,
		Game:              game,
		Upload:            upload,
		Build:             build,
		SecondsRun:        int64(legacyCave.SecondsRun),
		LastTouchedAt:     LastTouchedAt,
		InstalledAt:       InstalledAt,
	}

	runtime := ox.CurrentRuntime()
	consumer.Opf("Configuring cave for %s", runtime)
	verdict, err := manager.Configure(consumer, InstallFolder, runtime)
	if err != nil {
		return errors.WithStack(err)
	}
	cave.SetVerdict(verdict)
	cave.InstalledSize = verdict.TotalSize

	receipt := &bfs.Receipt{
		Game:   game,
		Upload: upload,
		Build:  build,
		// it's ok if len(files) == 0, we have a fallback
		Files: files,
		// that's ok too, we have a fallback
		InstallerName: "",
	}
	sc.addCave(cave, receipt)

	return nil
}

func (sc *scanContext) addCave(cave *models.Cave, receipt *bfs.Receipt) {
	sc.newByID[cave.ID] = &importedCave{
		cave:    cave,
		receipt: receipt,
	}

	messages.InstallLocationsScanYield.Notify(sc.rc, butlerd.InstallLocationsScanYieldNotification{
		Game: cave.Game,
	})
}

func (sc *scanContext) hasCave(caveID string) bool {
	if _, ok := sc.existingByID[caveID]; ok {
		return true
	}
	if _, ok := sc.newByID[caveID]; ok {
		return true
	}
	return false
}

func (sc *scanContext) hasCaveAtLoc(installLocationID string, installFolderName string) bool {
	for _, c := range sc.existingCaves {
		if c.InstallLocationID == installLocationID && c.InstallFolderName == installFolderName {
			return true
		}
	}

	for _, ic := range sc.newByID {
		c := ic.cave
		if c.InstallLocationID == installLocationID && c.InstallFolderName == installFolderName {
			return true
		}
	}
	return false
}

func (sc *scanContext) getInstallLocation(id string) *models.InstallLocation {
	for _, il := range sc.installLocations {
		if il.ID == id {
			return il
		}
	}
	return nil
}

func (sc *scanContext) queue(task *task) {
	sc.tasks = append(sc.tasks, task)
}

type legacyReceipt struct {
	Cave *legacyCave `json:"cave"`

	// Introduced in v0.12.0 (January 2016!!)
	// See https://github.com/itchio/itch/commit/1a550b52cb250b2b60b9c32b7f09fb8fc0bdc647#diff-2c5c1d50b675bd97c2b17e7fba3e5eeaR190
	Files []string `json:"files"`
}

type legacyCave struct {
	// These have been here since v0.14.0 at least,
	// see https://github.com/itchio/itch/blob/v0.14.0/appsrc/tasks/install.js#L42
	ID       string `json:"id"`
	GameID   int64  `json:"gameId"`
	UploadID int64  `json:"uploadId"`
	// see https://github.com/itchio/itch/blob/v0.14.0/appsrc/tasks/install.js#L103
	// and https://github.com/itchio/itch/commit/7f76cecbbae0a22bb574f8557b642acd24ee91d4#diff-0fcbef7d4100ec13581b447ef7050e7fL99
	BuildID int64 `json:"buildId"`

	// Introduced in v0.14.0
	// See https://github.com/itchio/itch/commit/62ac56a107ffe0a5d64e9574248a97761db0af8f#diff-edc50cd9cd6f10a0854f25aa2e3ea52dR35
	LastTouched float64 `json:"lastTouched"`
	SecondsRun  float64 `json:"secondsRun"`

	// Introduced in v18.0.0 as a `new Date()` (coerced to an RFC3339 string when
	// passed through JSON.stringify).
	// See https://github.com/itchio/itch/commit/eb046948d528053628a8b5dcd2860446223a8541#diff-0fcbef7d4100ec13581b447ef7050e7fR104
	// Changed to a number in v18.3.0,
	// See https://github.com/itchio/itch/commit/d3ad339a58c3763cef216015cce6c2b8084a89c2#diff-0fcbef7d4100ec13581b447ef7050e7fR114
	InstalledAt interface{} `json:"installedAt"`

	// Introduced in v18.5.0 (1 year 7 months ago at the time of this writing)
	// Prior to that, it was a mess, so let's just not import their caves.
	// See https://github.com/itchio/itch/commit/9cfc5c14184c506afff282ab50f19a7a774197a6#diff-b57d301bc67d78b1004baedc3472f13b
	PathScheme int `json:"pathScheme"`

	InstallLocation string `json:"installLocation"`
	InstallFolder   string `json:"installFolder"`
}

func fromJSTimestamp(timestamp float64) *time.Time {
	// javascript's Date.now() returns milliseconds since epoch
	// this assumes UTC timestamp
	secondsSinceEpoch := int64(timestamp / 1000.0)
	t := time.Unix(secondsSinceEpoch, 0)
	t = t.UTC()
	return &t
}

func fromJSDate(date string) *time.Time {
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return nil
	}
	t = t.UTC()
	return &t
}

type task struct {
	legacyCave *legacyCave
	files      []string
}
