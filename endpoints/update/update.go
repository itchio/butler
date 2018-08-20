package update

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/itchio/butler/installer/bfs"

	"github.com/arbovm/levenshtein"
	"github.com/itchio/ox"

	"github.com/itchio/butler/manager"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/cmd/operate/memorylogger"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

func Register(router *butlerd.Router) {
	messages.CheckUpdate.Register(router, CheckUpdate)
	messages.SnoozeCave.Register(router, SnoozeCave)
}

func CheckUpdate(rc *butlerd.RequestContext, params butlerd.CheckUpdateParams) (*butlerd.CheckUpdateResult, error) {
	startTime := time.Now()

	consumer := rc.Consumer
	res := &butlerd.CheckUpdateResult{}

	updateParams := checkUpdateCaveParams{
		rc:      rc,
		runtime: ox.CurrentRuntime(),
	}

	var caves []*models.Cave
	cond := builder.NewCond()
	if len(params.CaveIDs) > 0 {
		updateParams.ignoreSnooze = true

		var caveIDs []interface{}
		for _, cid := range params.CaveIDs {
			caveIDs = append(caveIDs, cid)
		}
		cond = builder.In("caves.id", caveIDs...)
	}

	search := hades.Search{}.OrderBy("caves.last_touched_at DESC")
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustSelect(conn, &caves, cond, search)
		models.PreloadCaves(conn, caves)
	})

	consumer.Infof("Looking for updates to %d items...", len(caves))
	consumer.Infof("...for runtime %s", updateParams.runtime)

	var doneCaves int

	// protects 'res' and 'doneCaves'
	var resultMutex sync.Mutex
	type taskSpec struct {
		cave *models.Cave
	}

	taskSpecs := make(chan taskSpec)
	workerDone := make(chan struct{})
	numWorkers := 4

	processOne := func(spec taskSpec) {
		defer func() {
			var err error
			if r := recover(); r != nil {
				if rErr, ok := r.(error); ok {
					err = errors.WithStack(rErr)
				} else {
					err = errors.Errorf("panic: %v", r)
				}
				consumer.Errorf("%+v", err)
				resultMutex.Lock()
				defer resultMutex.Unlock()
				res.Warnings = append(res.Warnings, fmt.Sprintf("%+v", err))
			}
		}()

		ml := memorylogger.New()
		update, err := checkUpdateCave(updateParams, ml.Consumer(), spec.cave)
		resultMutex.Lock()
		defer resultMutex.Unlock()

		doneCaves++
		consumer.Progress(float64(doneCaves) / float64(len(caves)))

		if err != nil {
			res.Warnings = append(res.Warnings, err.Error())
			consumer.Warnf("An update check failed: %+v", err)
			consumer.Warnf("Log follows ====================")
			ml.Copy(consumer)
			consumer.Warnf("Log ends here ==================")
		} else {
			if params.Verbose {
				ml.Copy(consumer)
			}
			if update != nil {
				res.Updates = append(res.Updates, update)
				err := messages.GameUpdateAvailable.Notify(rc, butlerd.GameUpdateAvailableNotification{
					Update: update,
				})
				if err != nil {
					consumer.Warnf("Could not send GameUpdateAvailable notification: %s", err.Error())
				}
			}
		}
	}

	work := func() {
		defer func() {
			workerDone <- struct{}{}
		}()

		for spec := range taskSpecs {
			processOne(spec)
		}
	}

	go func() {
		for _, cave := range caves {
			spec := taskSpec{
				cave: cave,
			}

			select {
			case taskSpecs <- spec:
				// good!
			case <-rc.Ctx.Done():
				close(taskSpecs)
				return
			}
		}

		close(taskSpecs)
	}()

	rc.StartProgress()
	for i := 0; i < numWorkers; i++ {
		go work()
	}

	for i := 0; i < numWorkers; i++ {
		<-workerDone
	}
	rc.EndProgress()

	consumer.Statf("Checked %d entries in %s", len(caves), time.Since(startTime))

	return res, nil
}

type checkUpdateCaveParams struct {
	ignoreSnooze bool
	rc           *butlerd.RequestContext
	runtime      *ox.Runtime
}

func checkUpdateCave(params checkUpdateCaveParams, consumer *state.Consumer, cave *models.Cave) (*butlerd.GameUpdate, error) {
	rc := params.rc
	runtime := params.runtime

	if cave.Pinned {
		consumer.Statf("Cave is pinned, skipping")
		return nil, nil
	}

	var access *operate.GameAccess
	rc.WithConn(func(conn *sqlite.Conn) {
		access = operate.AccessForGameID(conn, cave.GameID)
	})
	client := rc.Client(access.APIKey)

	if cave.Game == nil {
		consumer.Opf("Cave game is missing, trying to fetch it...")

		gameRes, err := client.GetGame(itchio.GetGameParams{
			GameID:      cave.GameID,
			Credentials: access.Credentials,
		})
		if err != nil {
			return nil, errors.Wrap(err, "Cave missing game, and error'd when fetching it")
		}

		cave.Game = gameRes.Game
		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, gameRes.Game)
		})
	}

	if cave.Upload == nil {
		consumer.Infof("Cave upload is missing, trying to fetch it...")
		consumer.Infof("(For game %s)", operate.GameToString(cave.Game))

		uploadRes, err := client.GetUpload(itchio.GetUploadParams{
			UploadID:    cave.UploadID,
			Credentials: access.Credentials,
		})
		if err != nil {
			cave.Upload = &itchio.Upload{
				ID: cave.UploadID,
			}
			consumer.Warnf("Can't retrieve upload %d: %v", cave.UploadID, err)
			consumer.Warnf("Continuing with a placeholder upload (this may give poor results)")
		} else {
			cave.Upload = uploadRes.Upload
			rc.WithConn(func(conn *sqlite.Conn) {
				models.MustSave(conn, uploadRes.Upload)
			})
		}
	}

	installFolder := rc.WithConnString(cave.GetInstallFolder)
	receipt, err := bfs.ReadReceipt(installFolder)
	if err != nil {
		return nil, err
	}

	if receipt != nil {
		// if we have a receipt, let's make sure it matches up
		// with the info we have in the database.
		// receipt info may be fresher if:
		//   * the game was updated by another copy of the itch app
		//     (for example, the beta, and we're now running stable)
		if receipt.Upload != nil && receipt.Upload.ID != cave.UploadID {
			consumer.Infof("Cave has:")
			operate.LogUpload(consumer, cave.Upload, cave.Build)
			consumer.Infof("But receipt has:")
			operate.LogUpload(consumer, receipt.Upload, receipt.Build)
			consumer.Infof("...fetching fresh info for receipt upload & build")

			uploadRes, err := client.GetUpload(itchio.GetUploadParams{
				UploadID:    receipt.Upload.ID,
				Credentials: access.Credentials,
			})
			if err != nil {
				cave.Upload = receipt.Upload
				consumer.Warnf("Can't retrieve upload %d: %v", receipt.Upload.ID, err)
				consumer.Warnf("Continuing with stored upload (this may give poor results)")
			} else {
				cave.Upload = uploadRes.Upload
			}

			if receipt.Build != nil {
				consumer.Infof("Also fetching info for build:")
				operate.LogBuild(consumer, receipt.Upload, receipt.Build)

				buildRes, err := client.GetBuild(itchio.GetBuildParams{
					BuildID:     receipt.Build.ID,
					Credentials: access.Credentials,
				})
				if err != nil {
					consumer.Warnf("Can't retrieve build %d: %v", receipt.Build.ID, err)
					consumer.Warnf("Continuing with stored build (this may give poor results)")
					cave.Build = receipt.Build
				} else {
					cave.Build = buildRes.Build
				}
			}

			rc.WithConn(func(conn *sqlite.Conn) {
				models.MustSave(conn, cave, hades.Assoc("Upload"), hades.Assoc("Build"))
			})
		}
	}

	consumer.Statf("Checking for updates to (%s)", operate.GameToString(cave.Game))

	if access.Credentials.DownloadKeyID > 0 {
		consumer.Infof("→ Has download key (game is owned)")
	} else {
		consumer.Infof("→ Searching without download key")
	}
	consumer.Infof("→ Last install operation at (%s)", cave.InstalledAt)

	consumer.Infof("→ Cached upload:")
	operate.LogUpload(consumer, cave.Upload, cave.Build)

	listUploadsRes, err := client.ListGameUploads(itchio.ListGameUploadsParams{
		GameID: cave.Game.ID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var currentUpload = cave.Upload
	var freshUpload *itchio.Upload
	var newerUploads []*itchio.Upload

	var referenceDate = cave.InstalledAt
	respectSnooze := !params.ignoreSnooze
	if respectSnooze && cave.SnoozedAt != nil {
		referenceDate = cave.SnoozedAt
		consumer.Infof("(Using reference snooze date (%s))", referenceDate)
	}

	for _, u := range listUploadsRes.Uploads {
		if u.ID == currentUpload.ID {
			consumer.Infof("✓ Installed upload still listed")
			freshUpload = u
			continue
		}

		if u.UpdatedAt == nil {
			consumer.Infof("↷ Skipping (nil updatedAt)")
			operate.LogUpload(consumer, u, u.Build)
			continue
		}

		if currentUpload.Type != "" && u.Type != currentUpload.Type {
			consumer.Infof("↷ Skipping (has type (%s) instead of (%s))", u.Type, currentUpload.Type)
			operate.LogUpload(consumer, u, u.Build)
			continue
		}

		if !moreRecentThan(u.UpdatedAt, referenceDate) {
			consumer.Infof("↷ Skipping (not more recent than reference date (%s))", referenceDate)
			operate.LogUpload(consumer, u, u.Build)
			continue
		}

		newerUploads = append(newerUploads, u)
	}

	// wharf updates
	if currentUpload.ChannelName != "" {
		consumer.Infof("We're currently on a wharf channel (%s)", currentUpload.ChannelName)

		if freshUpload != nil {
			consumer.Infof("...and our current upload still exists! Comparing builds...")
			if freshUpload.Build == nil {
				return nil, errors.New("We have a build installed but fresh upload has none. This shouldn't happen")
			}

			if freshUpload.Build.ID > cave.BuildID {
				consumer.Statf("↑ A more recent build (#%d) than ours (#%d) is available, it's an update!",
					freshUpload.Build.ID,
					cave.BuildID,
				)
				res := &butlerd.GameUpdate{
					CaveID: cave.ID,
					Game:   cave.Game,
					Direct: true,
				}

				res.Choices = append(res.Choices, &butlerd.GameUpdateChoice{
					Upload:     freshUpload,
					Build:      freshUpload.Build,
					Confidence: 1,
				})
				return res, nil
			}
			consumer.Infof("No direct update found, let's get fuzzy")
		}
	}

	// non-wharf updates
	if len(newerUploads) == 0 {
		consumer.Infof("No update found (no candidates)")
		return nil, nil
	}

	countBeforeNarrow := len(newerUploads)
	narrowDownResult := manager.NarrowDownUploads(consumer, cave.Game, newerUploads, runtime)
	newerUploads = narrowDownResult.Uploads
	consumer.Infof("→ %d uploads to consider (%d eliminated by narrow-down)", len(newerUploads), len(newerUploads)-countBeforeNarrow)

	if len(newerUploads) == 0 {
		consumer.Infof("No update found (no candidates)")
		return nil, nil
	}

	res := &butlerd.GameUpdate{
		CaveID: cave.ID,
		Game:   cave.Game,
		Direct: false,
	}

	consumer.Infof("→ Considered uploads:")
	for _, u := range newerUploads {
		operate.LogUpload(consumer, u, u.Build)

		var lhs, rhs string
		if u.DisplayName != "" && currentUpload.DisplayName != "" {
			lhs = u.DisplayName
			rhs = currentUpload.DisplayName
		} else {
			lhs = u.Filename
			rhs = currentUpload.Filename
		}
		dist := levenshtein.Distance(lhs, rhs)
		consumer.Infof("(distance of %d between (%s) and (%s))", dist, lhs, rhs)
		confidence := 1.0 / (1.0 + float64(dist)/10.0)

		choice := &butlerd.GameUpdateChoice{
			Upload:     u,
			Build:      u.Build,
			Confidence: confidence,
		}
		res.Choices = append(res.Choices, choice)
	}

	consumer.Infof("Sorting by confidence...")
	sort.Slice(res.Choices, func(i, j int) bool {
		ii := res.Choices[i]
		jj := res.Choices[j]
		// sort from most confidence to least confidence
		return ii.Confidence > jj.Confidence
	})

	consumer.Infof("→ Final draft:")
	for _, c := range res.Choices {
		operate.LogUpload(consumer, c.Upload, c.Build)
	}

	return res, nil
}

func moreRecentThan(lhs *time.Time, rhs *time.Time) bool {
	if lhs == nil {
		// always less recent if we lack a timestamp
		return false
	}

	if rhs == nil {
		// always more recent than something that lacks a timestamp
		return true
	}

	return (*lhs).After(*rhs)
}

func SnoozeCave(rc *butlerd.RequestContext, params butlerd.SnoozeCaveParams) (*butlerd.SnoozeCaveResult, error) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	cave := models.CaveByID(conn, params.CaveID)
	if cave == nil {
		return nil, errors.Errorf("No such cave (%s)", params.CaveID)
	}

	now := time.Now().UTC()
	cave.SnoozedAt = &now
	cave.Save(conn)

	return &butlerd.SnoozeCaveResult{}, nil
}
