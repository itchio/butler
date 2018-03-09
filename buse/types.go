package buse

import (
	"time"

	itchio "github.com/itchio/go-itchio"
)

// must be kept in sync with clients, see for example
// https://github.com/itchio/node-butler

//----------------------------------------------------------------------
// Version
//----------------------------------------------------------------------

// Retrieves the version of the butler instance the client
// is connected to.
//
// This endpoint is meant to gather information when reporting
// issues, rather than feature sniffing. Conforming clients should
// automatically download new versions of butler, see the **Updating** section.
//
// @name Version.Get
// @category Utilities
// @tags Offline
// @caller client
type VersionGetParams struct{}

type VersionGetResult struct {
	// Something short, like `v8.0.0`
	Version string `json:"version"`

	// Something long, like `v8.0.0, built on Aug 27 2017 @ 01:13:55, ref d833cc0aeea81c236c81dffb27bc18b2b8d8b290`
	VersionString string `json:"versionString"`
}

//----------------------------------------------------------------------
// Profile
//----------------------------------------------------------------------

// Lists remembered profiles
//
// @name Profile.List
// @category Profile
// @caller client
type ProfileListParams struct {
}

type ProfileListResult struct {
	// A list of remembered profiles
	Profiles []*Profile `json:"profiles"`
}

// Represents a user for which we have profile information,
// ie. that we can connect as, etc.
type Profile struct {
	// itch.io user ID, doubling as profile ID
	ID int64 `json:"id"`

	// Timestamp the user last connected at (to the client)
	LastConnected time.Time `json:"lastConnected"`

	// User information
	User *itchio.User `json:"user"`
}

// Add a new profile by password login
//
// @name Profile.LoginWithPassword
// @category Profile
// @caller client
type ProfileLoginWithPasswordParams struct {
	// The username (or e-mail) to use for login
	Username string `json:"username"`

	// The password to use
	Password string `json:"password"`
}

type ProfileLoginWithPasswordResult struct {
	// Information for the new profile, now remembered
	Profile *Profile `json:"profile"`

	// Profile cookie for website
	Cookie map[string]string `json:"cookie"`
}

// Ask the user to solve a captcha challenge
// Sent during @@ProfileLoginWithPasswordParams if certain
// conditions are met.
//
// @name Profile.RequestCaptcha
// @category Profile
// @caller server
type ProfileRequestCaptchaParams struct {
	// Address of page containing a recaptcha widget
	RecaptchaURL string `json:"recaptchaUrl"`
}

type ProfileRequestCaptchaResult struct {
	// The response given by recaptcha after it's been filled
	RecaptchaResponse string `json:"recaptchaResponse"`
}

// Ask the user to provide a TOTP token.
// Sent during @@ProfileLoginWithPasswordParams if the user has
// two-factor authentication enabled.
//
// @name Profile.RequestTOTP
// @category Profile
// @caller server
type ProfileRequestTOTPParams struct {
}

type ProfileRequestTOTPResult struct {
	// The TOTP code entered by the user
	Code string `json:"code"`
}

// Use saved login credentials to validate a profile.
//
// @name Profile.UseSavedLogin
// @category Profile
// @caller client
type ProfileUseSavedLoginParams struct {
	ProfileID int64 `json:"profileId"`
}

type ProfileUseSavedLoginResult struct {
	// Information for the now validated profile
	Profile *Profile `json:"profile"`
}

// Forgets a remembered profile - it won't appear in the
// @@ProfileListParams results anymore.
//
// @name Profile.Forget
// @category Profile
// @caller client
type ProfileForgetParams struct {
	ProfileID int64 `json:"profileId"`
}

type ProfileForgetResult struct {
	// True if the profile did exist (and was successfully forgotten)
	Success bool `json:"success"`
}

//----------------------------------------------------------------------
// Fetch
//----------------------------------------------------------------------

// Fetches information for an itch.io game.
//
// Sends @@FetchGameYieldNotification twice at most: first from cache,
// second from API if we're online.
//
// @name Fetch.Game
// @category Fetch
// @caller client
type FetchGameParams struct {
	// Profile to use to fetch game
	ProfileID int64 `json:"profileId"`
	// Identifier of game to look for
	GameID int64 `json:"gameId"`
}

// Sent during @@FetchGameParams whenever a result is
// available.
//
// @name Fetch.Game.Yield
// @category Fetch
type FetchGameYieldNotification struct {
	// Current result for game fetching (from local DB, or API, etc.)
	Game *itchio.Game `json:"game"`
}

type FetchGameResult struct {
}

// Fetches information about a collection and the games it
// contains.
//
// Sends @@FetchCollectionYieldNotification.
//
// @name Fetch.Collection
// @category Fetch
// @caller client
type FetchCollectionParams struct {
	// Profile to use to fetch game
	ProfileID int64 `json:"profileId"`
	// Identifier of the collection to look for
	CollectionID int64 `json:"collectionId"`
}

// Contains general info about a collection
//
// @name Fetch.Collection.Yield
// @category Fetch
type FetchCollectionYieldNotification struct {
	Collection *itchio.Collection `json:"collection"`
}

// Association between a @@Game and a @@Collection
// @category Fetch
type CollectionGame struct {
	// Position in collection, use if you want to display them in the
	// canonical itch.io order
	Position int64        `json:"position"`
	Game     *itchio.Game `json:"game"`
}

type FetchCollectionResult struct {
}

// @name Fetch.ProfileCollections
// @category Fetch
// @caller client
type FetchProfileCollectionsParams struct {
	// Profile to use to fetch game
	ProfileID int64 `json:"profileId"`
}

// Sent during @@FetchProfileCollectionsParams whenever new info is
// available.
//
// @name Fetch.ProfileCollections.Yield
// @category Fetch
type FetchProfileCollectionsYieldNotification struct {
	Offset int64                `json:"offset"`
	Total  int64                `json:"total"`
	Items  []*itchio.Collection `json:"items"`
}

type FetchProfileCollectionsResult struct {
}

// @name Fetch.ProfileGames
// @category Fetch
// @caller client
type FetchProfileGamesParams struct {
	// Profile to use to fetch game
	ProfileID int64 `json:"profileId"`
}

type ProfileGame struct {
	Game *itchio.Game `json:"game,omitempty"`
	User *itchio.User `json:"user,omitempty"`

	// Position on profile, from 0 to N
	Position int64 `json:"position"`

	ViewsCount     int64 `json:"viewsCount"`
	DownloadsCount int64 `json:"downloadsCount"`
	PurchasesCount int64 `json:"purchasesCount"`

	Published bool `json:"published"`
}

// @name Fetch.ProfileGames.Yield
// @category Fetch
type FetchProfileGamesYieldNotification struct {
	Offset int64          `json:"offset"`
	Total  int64          `json:"total"`
	Items  []*ProfileGame `json:"items"`
}

type FetchProfileGamesResult struct {
}

// @name Fetch.ProfileOwnedKeys
// @category Fetch
// @caller client
type FetchProfileOwnedKeysParams struct {
	// Profile to use to fetch game
	ProfileID int64 `json:"profileId"`
}

// @name Fetch.ProfileOwnedKeys.Yield
// @category Fetch
type FetchProfileOwnedKeysYieldNotification struct {
	Offset int64                 `json:"offset"`
	Total  int64                 `json:"total"`
	Items  []*itchio.DownloadKey `json:"items"`
}

type FetchProfileOwnedKeysResult struct {
}

// @name Fetch.Commons
// @category Fetch
// @caller client
type FetchCommonsParams struct{}

type FetchCommonsResult struct {
	DownloadKeys     []*DownloadKeySummary     `json:"downloadKeys"`
	Caves            []*CaveSummary            `json:"caves"`
	InstallLocations []*InstallLocationSummary `json:"installLocations"`
}

type DownloadKeySummary struct {
	// Site-wide unique identifier generated by itch.io
	ID int64 `json:"id"`

	// Identifier of the game to which this download key grants access
	GameID int64 `json:"gameId"`

	// Date this key was created at (often coincides with purchase time)
	CreatedAt time.Time `json:"createdAt"`
}

type CaveSummary struct {
	ID string `json:"id"`

	GameID int64 `json:"gameId"`

	LastTouchedAt *time.Time `json:"lastTouchedAt"`
	SecondsRun    int64      `json:"secondsRun"`
	InstalledSize int64      `json:"installedSize"`
}

type Cave struct {
	ID string `json:"id"`

	Game   *itchio.Game   `json:"game"`
	Upload *itchio.Upload `json:"upload"`
	Build  *itchio.Build  `json:"build"`

	Stats       *CaveStats       `json:"stats"`
	InstallInfo *CaveInstallInfo `json:"installInfo"`
}

type CaveStats struct {
	InstalledAt   *time.Time `json:"installedAt"`
	LastTouchedAt *time.Time `json:"lastTouchedAt"`
	SecondsRun    int64      `json:"secondsRun"`
}

type CaveInstallInfo struct {
	InstalledSize   int64  `json:"installedSize"`
	InstallLocation string `json:"installLocation"`
	InstallFolder   string `json:"installFolder"`
}

type InstallLocationSummary struct {
	InstallLocation string `json:"installLocation"`
	Size            int64  `json:"size"`
}

// Retrieve info for all caves.
//
// @name Fetch.Caves
// @category Fetch
// @caller client
type FetchCavesParams struct {
}

type FetchCavesResult struct {
	Caves []*Cave `json:"caves"`
}

// Retrieve info on a cave by ID.
//
// @name Fetch.Cave
// @category Fetch
// @caller client
type FetchCaveParams struct {
	CaveID string `json:"caveId"`
}

type FetchCaveResult struct {
	Cave *Cave `json:"cave"`
}

// Retrieve all caves for a given game.
//
// @name Fetch.CavesByGameID
// @category Fetch
// @caller client
type FetchCavesByGameIDParams struct {
	GameID int64 `json:"gameId"`
}

type FetchCavesByGameIDResult struct {
	Caves []*Cave `json:"caves"`
}

// Retrieve all caves installed to a given location.
//
// @name Fetch.CavesByInstallLocationID
// @category Fetch
// @caller client
type FetchCavesByInstallLocationIDParams struct {
	InstallLocationID string `json:"installLocationId"`
}

type FetchCavesByInstallLocationIDResult struct {
	InstallLocationPath string  `json:"installLocationPath"`
	InstallLocationSize int64   `json:"installLocationSize"`
	Caves               []*Cave `json:"caves"`
}

//----------------------------------------------------------------------
// Game
//----------------------------------------------------------------------

// Finds uploads compatible with the current runtime, for a given game.
//
// @name Game.FindUploads
// @category Install
// @caller client
type GameFindUploadsParams struct {
	// Which game to find uploads for
	Game *itchio.Game `json:"game"`
}

type GameFindUploadsResult struct {
	// A list of uploads that were found to be compatible.
	Uploads []*itchio.Upload `json:"uploads"`
}

//----------------------------------------------------------------------
// Install
//----------------------------------------------------------------------

// Queues an install operation to be later performed
// via @@InstallPerformParams.
//
// @name Install.Queue
// @category Install
// @caller client
type InstallQueueParams struct {
	// ID of the cave to perform the install for.
	// If not specified, will create a new cave.
	// @optional
	CaveID string `json:"caveId"`

	// If unspecified, will default to 'install'
	// @optional
	Reason DownloadReason `json:"reason"`

	// If CaveID is not specified, ID of an install location
	// to install to.
	// @optional
	InstallLocationID string `json:"installLocationId"`

	// If set, InstallFolder can be set and no cave
	// record will be read or modified
	// @optional
	NoCave bool `json:"noCave"`

	// When NoCave is set, exactly where to install
	// @optional
	InstallFolder string `json:"installFolder"`

	// Which game to install.
	//
	// If unspecified and caveId is specified, the same game will be used.
	// @optional
	Game *itchio.Game `json:"game"`

	// Which upload to install.
	//
	// If unspecified and caveId is specified, the same upload will be used.
	// @optional
	Upload *itchio.Upload `json:"upload"`

	// Which build to install
	//
	// If unspecified and caveId is specified, the same build will be used.
	// @optional
	Build *itchio.Build `json:"build"`

	// If true, do not run windows installers, just extract
	// whatever to the install folder.
	// @optional
	IgnoreInstallers bool `json:"ignoreInstallers,omitempty"`

	// A folder that butler can use to store temporary files, like
	// partial downloads, checkpoint files, etc.
	// @optional
	StagingFolder string `json:"stagingFolder"`

	// If set, and the install operation is successfully disambiguated,
	// will queue it as a download for butler to drive.
	// See @@DownloadsDriveParams.
	// @optional
	QueueDownload bool `json:"queueDownload"`
}

type InstallQueueResult struct {
	ID            string         `json:"id"`
	Reason        DownloadReason `json:"reason"`
	CaveID        string         `json:"caveId"`
	Game          *itchio.Game   `json:"game"`
	Upload        *itchio.Upload `json:"upload"`
	Build         *itchio.Build  `json:"build"`
	InstallFolder string         `json:"installFolder"`
	StagingFolder string         `json:"stagingFolder"`
}

// Perform an install that was previously queued via
// @@InstallQueueParams.
//
// Can be cancelled by passing the same `ID` to @@InstallCancelParams.
//
// @name Install.Perform
// @category Install
// @tags Cancellable
// @caller client
type InstallPerformParams struct {
	// ID that can be later used in @@InstallCancelParams
	ID string `json:"id"`

	// The folder turned by @@InstallQueueParams
	StagingFolder string `json:"stagingFolder"`
}

type InstallPerformResult struct{}

// Attempt to gracefully cancel an ongoing operation.
//
// @name Install.Cancel
// @category Install
// @caller client
type InstallCancelParams struct {
	// The UUID of the task to cancel, as passed to @@OperationStartParams
	ID string `json:"id"`
}

type InstallCancelResult struct{}

// UninstallParams contains all the parameters needed to perform
// an uninstallation for a game via @@OperationStartParams.
//
// @name Uninstall.Perform
// @category Install
// @caller client
type UninstallPerformParams struct {
	// The cave to uninstall
	CaveID string `json:"caveId"`
}

type UninstallPerformResult struct{}

// Prepare to queue a version switch. The client will
// receive an @@InstallVersionSwitchPickParams.
//
// @name Install.VersionSwitch.Queue
// @category Install
// @caller client
type InstallVersionSwitchQueueParams struct {
	// The cave to switch to a different version
	CaveID string `json:"caveId"`
}

type InstallVersionSwitchQueueResult struct {
}

// Let the user pick which version to switch to.
//
// @category Install
// @caller server
type InstallVersionSwitchPickParams struct {
	Cave   *Cave           `json:"cave"`
	Upload *itchio.Upload  `json:"upload"`
	Builds []*itchio.Build `json:"builds"`
}

type InstallVersionSwitchPickResult struct {
	// A negative index aborts the version switch
	Index int64 `json:"index"`
}

// GameCredentials contains all the credentials required to make API requests
// including the download key if any.
type GameCredentials struct {
	// Defaults to `https://itch.io`
	// @optional
	Server string `json:"server"`
	// A valid itch.io API key
	APIKey string `json:"apiKey"`
	// A download key identifier, or 0 if no download key is available
	// @optional
	DownloadKey int64 `json:"downloadKey"`
}

// Asks the user to pick between multiple available uploads
//
// @category Install
// @tags Dialog
// @caller server
type PickUploadParams struct {
	// An array of upload objects to choose from
	Uploads []*itchio.Upload `json:"uploads"`
}

type PickUploadResult struct {
	// The index (in the original array) of the upload that was picked,
	// or a negative value to cancel.
	Index int64 `json:"index"`
}

// Sent periodically during @@InstallPerformParams to inform on the current state of an install
//
// @name Progress
// @category Install
type ProgressNotification struct {
	// An overall progress value between 0 and 1
	Progress float64 `json:"progress"`
	// Estimated completion time for the operation, in seconds (floating)
	ETA float64 `json:"eta"`
	// Network bandwidth used, in bytes per second (floating)
	BPS float64 `json:"bps"`
}

// @category Install
type TaskReason string

const (
	// Task was started for an install operation
	TaskReasonInstall TaskReason = "install"
	// Task was started for an uninstall operation
	TaskReasonUninstall TaskReason = "uninstall"
)

// @category Install
type TaskType string

const (
	// We're fetching files from a remote server
	TaskTypeDownload TaskType = "download"
	// We're running an installer
	TaskTypeInstall TaskType = "install"
	// We're running an uninstaller
	TaskTypeUninstall TaskType = "uninstall"
	// We're applying some patches
	TaskTypeUpdate TaskType = "update"
	// We're healing from a signature and heal source
	TaskTypeHeal TaskType = "heal"
)

// Each operation is made up of one or more tasks. This notification
// is sent during @@OperationStartParams whenever a specific task starts.
//
// @category Install
type TaskStartedNotification struct {
	// Why this task was started
	Reason TaskReason `json:"reason"`
	// Is this task a download? An install?
	Type TaskType `json:"type"`
	// The game this task is dealing with
	Game *itchio.Game `json:"game"`
	// The upload this task is dealing with
	Upload *itchio.Upload `json:"upload"`
	// The build this task is dealing with (if any)
	Build *itchio.Build `json:"build,omitempty"`
	// Total size in bytes
	TotalSize int64 `json:"totalSize,omitempty"`
}

// Sent during @@OperationStartParams whenever a task succeeds for an operation.
//
// @category Install
type TaskSucceededNotification struct {
	Type TaskType `json:"type"`
	// If the task installed something, then this contains
	// info about the game, upload, build that were installed
	InstallResult *InstallResult `json:"installResult,omitempty"`
}

// What was installed by a subtask of @@OperationStartParams.
//
// See @@TaskSucceededNotification.
//
// @category Install
// @kind type
type InstallResult struct {
	// The game we installed
	Game *itchio.Game `json:"game"`
	// The upload we installed
	Upload *itchio.Upload `json:"upload"`
	// The build we installed
	// @optional
	Build *itchio.Build `json:"build"`
	// TODO: verdict ?
}

//----------------------------------------------------------------------
// Downloads
//----------------------------------------------------------------------

// Queue a download that will be performed later by
// @@DownloadsDriveParams.
//
// @name Downloads.Queue
// @category Downloads
// @caller client
type DownloadsQueueParams struct {
	Item *InstallQueueResult `json:"item"`
}

type DownloadsQueueResult struct {
}

// Put a download on top of the queue.
//
// @name Downloads.Prioritize
// @category Downloads
// @caller client
type DownloadsPrioritizeParams struct {
	DownloadID string `json:"downloadId"`
}

type DownloadsPrioritizeResult struct {
}

// List all known downloads.
//
// @name Downloads.List
// @category Downloads
// @caller client
type DownloadsListParams struct {
}

type DownloadsListResult struct {
	Downloads []*Download `json:"downloads"`
}

// Removes all finished downloads from the queue.
//
// @name Downloads.ClearFinished
// @category Downloads
// @caller client
type DownloadsClearFinishedParams struct {
}

type DownloadsClearFinishedResult struct {
	// The number of removed downloads
	RemovedCount int64 `json:"removedCount"`
}

// Drive downloads, which is: perform them one at a time,
// until they're all finished.
//
// @name Downloads.Drive
// @category Downloads
// @caller client
type DownloadsDriveParams struct{}

type DownloadsDriveResult struct{}

// Stop driving downloads gracefully.
//
// @name Downloads.Drive.Cancel
// @category Downloads
// @caller client
type DownloadsDriveCancelParams struct{}

type DownloadsDriveCancelResult struct{}

// @name Downloads.Drive.Progress
type DownloadsDriveProgressNotification struct {
	Download *Download         `json:"download"`
	Progress *DownloadProgress `json:"progress"`
}

// @name Downloads.Drive.Finished
type DownloadsDriveFinishedNotification struct {
	Download *Download `json:"download"`
}

type DownloadReason string

const (
	DownloadReasonInstall       DownloadReason = "install"
	DownloadReasonReinstall     DownloadReason = "reinstall"
	DownloadReasonUpdate        DownloadReason = "update"
	DownloadReasonVersionSwitch DownloadReason = "version-switch"
)

// Represents a download queued, which will be
// performed whenever @@DownloadsDriveParams is called.
type Download struct {
	ID            string         `json:"id"`
	Error         *string        `json:"error"`
	Reason        DownloadReason `json:"reason"`
	Position      int64          `json:"position"`
	CaveID        string         `json:"caveId"`
	Game          *itchio.Game   `json:"game"`
	Upload        *itchio.Upload `json:"upload"`
	Build         *itchio.Build  `json:"build"`
	StartedAt     *time.Time     `json:"startedAt"`
	FinishedAt    *time.Time     `json:"finishedAt"`
	StagingFolder string         `json:"stagingFolder"`
}

type DownloadProgress struct {
	Stage    string  `json:"stage"`
	Progress float64 `json:"progress"`
	ETA      float64 `json:"eta"`
	BPS      float64 `json:"bps"`
}

// Retries a download that has errored
//
// @name Downloads.Retry
// @category Downloads
// @caller client
type DownloadsRetryParams struct {
	DownloadID string `json:"downloadId"`
}

type DownloadsRetryResult struct{}

// Attempts to discard a download
//
// @name Downloads.Discard
// @category Downloads
// @caller client
type DownloadsDiscardParams struct {
	DownloadID string `json:"downloadId"`
}

type DownloadsDiscardResult struct{}

//----------------------------------------------------------------------
// CheckUpdate
//----------------------------------------------------------------------

// Looks for one or more game updates.
//
// Updates found are regularly sent via @@GameUpdateAvailableNotification, and
// then all at once in the result.
//
// @category Update
// @caller client
type CheckUpdateParams struct {
	// A list of items, each of it will be checked for updates
	Items []*CheckUpdateItem `json:"items"`
}

// @category Update
type CheckUpdateItem struct {
	// An UUID generated by the client, which allows it to map back the
	// results to its own items.
	ItemID string `json:"itemId"`
	// Timestamp of the last successful install operation
	InstalledAt time.Time `json:"installedAt"`
	// Game for which to look for an update
	Game *itchio.Game `json:"game"`
	// Currently installed upload
	Upload *itchio.Upload `json:"upload"`
	// Currently installed build
	Build *itchio.Build `json:"build,omitempty"`
}

type CheckUpdateResult struct {
	// Any updates found (might be empty)
	Updates []*GameUpdate `json:"updates"`
	// Warnings messages logged while looking for updates
	Warnings []string `json:"warnings"`
}

// Sent during @@CheckUpdateParams, every time butler
// finds an update for a game. Can be safely ignored if displaying
// updates as they are found is not a requirement for the client.
//
// @category Update
// @tags Optional
type GameUpdateAvailableNotification struct {
	Update *GameUpdate `json:"update"`
}

// Describes an available update for a particular game install.
//
// @category Update
type GameUpdate struct {
	// Identifier originally passed in CheckUpdateItem
	ItemID string `json:"itemId"`
	// Game we found an update for
	Game *itchio.Game `json:"game"`
	// Upload to be installed
	Upload *itchio.Upload `json:"upload"`
	// Build to be installed (may be nil)
	Build *itchio.Build `json:"build"`
}

//----------------------------------------------------------------------
// Launch
//----------------------------------------------------------------------

// Attempt to launch an installed game.
//
// @category Launch
// @caller client
type LaunchParams struct {
	// The ID of the cave to launch
	CaveID string `json:"caveId"`

	// The directory to use to store installer files for prerequisites
	PrereqsDir string `json:"prereqsDir"`
	// Force installing all prerequisites, even if they're already marked as installed
	// @optional
	ForcePrereqs bool `json:"forcePrereqs,omitempty"`

	// Enable sandbox (regardless of manifest opt-in)
	Sandbox bool `json:"sandbox,omitempty"`
}

type LaunchResult struct {
}

// Sent during @@LaunchParams, when the game is configured, prerequisites are installed
// sandbox is set up (if enabled), and the game is actually running.
//
// @category Launch
type LaunchRunningNotification struct{}

// Sent during @@LaunchParams, when the game has actually exited.
//
// @category Launch
type LaunchExitedNotification struct{}

// Sent during @@LaunchParams, ask the user to pick a manifest action to launch.
//
// See [itch app manifests](https://itch.io/docs/itch/integrating/manifest.html).
//
// @tags Dialogs
// @category Launch
// @caller server
type PickManifestActionParams struct {
	// A list of actions to pick from. Must be shown to the user in the order they're passed.
	Actions []*Action `json:"actions"`
}

type PickManifestActionResult struct {
	// Index of action picked by user, or negative if aborting
	Index int `json:"index"`
}

// Ask the client to perform a shell launch, ie. open an item
// with the operating system's default handler (File explorer).
//
// Sent during @@LaunchParams.
//
// @category Launch
// @caller server
type ShellLaunchParams struct {
	// Absolute path of item to open, e.g. `D:\\Games\\Itch\\garden\\README.txt`
	ItemPath string `json:"itemPath"`
}

type ShellLaunchResult struct {
}

// Ask the client to perform an HTML launch, ie. open an HTML5
// game, ideally in an embedded browser.
//
// Sent during @@LaunchParams.
//
// @category Launch
// @caller server
type HTMLLaunchParams struct {
	// Absolute path on disk to serve
	RootFolder string `json:"rootFolder"`
	// Path of index file, relative to root folder
	IndexPath string `json:"indexPath"`

	// Command-line arguments, to pass as `global.Itch.args`
	Args []string `json:"args"`
	// Environment variables, to pass as `global.Itch.env`
	Env map[string]string `json:"env"`
}

type HTMLLaunchResult struct {
}

// Ask the client to perform an URL launch, ie. open an address
// with the system browser or appropriate.
//
// Sent during @@LaunchParams.
//
// @category Launch
// @caller server
type URLLaunchParams struct {
	// URL to open, e.g. `https://itch.io/community`
	URL string `json:"url"`
}

type URLLaunchResult struct{}

// Ask the user to allow sandbox setup. Will be followed by
// a UAC prompt (on Windows) or a pkexec dialog (on Linux) if
// the user allows.
//
// Sent during @@LaunchParams.
//
// @category Launch
// @tags Dialogs
// @caller server
type AllowSandboxSetupParams struct{}

type AllowSandboxSetupResult struct {
	// Set to true if user allowed the sandbox setup, false otherwise
	Allow bool `json:"allow"`
}

// Sent during @@LaunchParams, when some prerequisites are about to be installed.
//
// This is a good time to start showing a UI element with the state of prereq
// tasks.
//
// Updates are regularly provided via @@PrereqsTaskStateNotification.
//
// @category Launch
type PrereqsStartedNotification struct {
	// A list of prereqs that need to be tended to
	Tasks map[string]*PrereqTask `json:"tasks"`
}

// Information about a prerequisite task.
//
// @category Launch
type PrereqTask struct {
	// Full name of the prerequisite, for example: `Microsoft .NET Framework 4.6.2`
	FullName string `json:"fullName"`
	// Order of task in the list. Respect this order in the UI if you want consistent progress indicators.
	Order int `json:"order"`
}

// Current status of a prerequisite task
//
// Sent during @@LaunchParams, after @@PrereqsStartedNotification, repeatedly
// until all prereq tasks are done.
//
// @category Launch
type PrereqsTaskStateNotification struct {
	// Short name of the prerequisite task (e.g. `xna-4.0`)
	Name string `json:"name"`
	// Current status of the prereq
	Status PrereqStatus `json:"status"`
	// Value between 0 and 1 (floating)
	Progress float64 `json:"progress"`
	// ETA in seconds (floating)
	ETA float64 `json:"eta"`
	// Network bandwidth used in bytes per second (floating)
	BPS float64 `json:"bps"`
}

// @category Launch
type PrereqStatus string

const (
	// Prerequisite has not started downloading yet
	PrereqStatusPending PrereqStatus = "pending"
	// Prerequisite is currently being downloaded
	PrereqStatusDownloading PrereqStatus = "downloading"
	// Prerequisite has been downloaded and is pending installation
	PrereqStatusReady PrereqStatus = "ready"
	// Prerequisite is currently installing
	PrereqStatusInstalling PrereqStatus = "installing"
	// Prerequisite was installed (successfully or not)
	PrereqStatusDone PrereqStatus = "done"
)

// Sent during @@LaunchParams, when all prereqs have finished installing (successfully or not)
//
// After this is received, it's safe to close any UI element showing prereq task state.
//
// @category Launch
type PrereqsEndedNotification struct {
}

// Sent during @@LaunchParams, when one or more prerequisites have failed to install.
// The user may choose to proceed with the launch anyway.
//
// @category Launch
// @caller server
type PrereqsFailedParams struct {
	// Short error
	Error string `json:"error"`
	// Longer error (to include in logs)
	ErrorStack string `json:"errorStack"`
}

type PrereqsFailedResult struct {
	// Set to true if the user wants to proceed with the launch in spite of the prerequisites failure
	Continue bool `json:"continue"`
}

//----------------------------------------------------------------------
// CleanDownloads
//----------------------------------------------------------------------

// Look for folders we can clean up in various download folders.
// This finds anything that doesn't correspond to any current downloads
// we know about.
//
// @name CleanDownloads.Search
// @category Clean Downloads
// @caller client
type CleanDownloadsSearchParams struct {
	// A list of folders to scan for potential subfolders to clean up
	Roots []string `json:"roots"`
	// A list of subfolders to not consider when cleaning
	// (staging folders for in-progress downloads)
	Whitelist []string `json:"whitelist"`
}

// @category Clean Downloads
type CleanDownloadsSearchResult struct {
	// Entries we found that could use some cleaning (with path and size information)
	Entries []*CleanDownloadsEntry `json:"entries"`
}

// @category Clean Downloads
type CleanDownloadsEntry struct {
	// The complete path of the file or folder we intend to remove
	Path string `json:"path"`
	// The size of the folder or file, in bytes
	Size int64 `json:"size"`
}

// Remove the specified entries from disk, freeing up disk space.
//
// @name CleanDownloads.Apply
// @category Clean Downloads
// @caller client
type CleanDownloadsApplyParams struct {
	Entries []*CleanDownloadsEntry `json:"entries"`
}

// @category Clean Downloads
type CleanDownloadsApplyResult struct{}

//----------------------------------------------------------------------
// Misc.
//----------------------------------------------------------------------

// Sent any time butler needs to send a log message. The client should
// relay them in their own stdout / stderr, and collect them so they
// can be part of an issue report if something goes wrong.
type LogNotification struct {
	// Level of the message (`info`, `warn`, etc.)
	Level LogLevel `json:"level"`
	// Contents of the message.
	//
	// Note: logs may contain non-ASCII characters, or even emojis.
	Message string `json:"message"`
}

type LogLevel string

const (
	// Hidden from logs by default, noisy
	LogLevelDebug LogLevel = "debug"
	// Just thinking out loud
	LogLevelInfo LogLevel = "info"
	// We're continuing, but we're not thrilled about it
	LogLevelWarning LogLevel = "warning"
	// We're eventually going to fail loudly
	LogLevelError LogLevel = "error"
)

// Test request: asks butler to double a number twice.
// First by calling @@TestDoubleParams, then by
// returning the result of that call doubled.
//
// Use that to try out your JSON-RPC 2.0 over TCP implementation.
//
// @name Test.DoubleTwice
// @category Test
// @caller client
type TestDoubleTwiceParams struct {
	// The number to quadruple
	Number int64 `json:"number"`
}

// @category Test
type TestDoubleTwiceResult struct {
	// The input, quadrupled
	Number int64 `json:"number"`
}

// Test request: return a number, doubled. Implement that to
// use @@TestDoubleTwiceParams in your testing.
//
// @name Test.Double
// @category Test
// @caller server
type TestDoubleParams struct {
	// The number to double
	Number int64 `json:"number"`
}

// Result for Test.Double
type TestDoubleResult struct {
	// The number, doubled
	Number int64 `json:"number"`
}

type ItchPlatform string

const (
	ItchPlatformOSX     ItchPlatform = "osx"
	ItchPlatformWindows ItchPlatform = "windows"
	ItchPlatformLinux   ItchPlatform = "linux"
	ItchPlatformUnknown ItchPlatform = "unknown"
)

// Buse JSON-RPC 2.0 error codes
type Code int64

const (
	// An operation was cancelled gracefully
	CodeOperationCancelled Code = 499
	// An operation was aborted by the user
	CodeOperationAborted Code = 410
)

//==================================
// Manifests
//==================================

// A Manifest describes prerequisites (dependencies) and actions that
// can be taken while launching a game.
type Manifest struct {
	// Actions are a list of options to give the user when launching a game.
	Actions []*Action `json:"actions"`

	// Prereqs describe libraries or frameworks that must be installed
	// prior to launching a game
	Prereqs []*Prereq `json:"prereqs,omitempty"`
}

// An Action is a choice for the user to pick when launching a game.
//
// see https://itch.io/docs/itch/integrating/manifest.html
type Action struct {
	// human-readable or standard name
	Name string `json:"name"`

	// file path (relative to manifest or absolute), URL, etc.
	Path string `json:"path"`

	// icon name (see static/fonts/icomoon/demo.html, don't include `icon-` prefix)
	Icon string `json:"icon,omitempty"`

	// command-line arguments
	Args []string `json:"args,omitempty"`

	// sandbox opt-in
	Sandbox bool `json:"sandbox,omitempty"`

	// requested API scope
	Scope string `json:"scope,omitempty"`

	// don't redirect stdout/stderr, open in new console window
	Console bool `json:"console,omitempty"`

	// platform to restrict this action too
	Platform ItchPlatform `json:"platform,omitempty"`

	// localized action name
	Locales map[string]*ActionLocale `json:"locales,omitempty"`
}

type Prereq struct {
	// A prerequisite to be installed, see <https://itch.io/docs/itch/integrating/prereqs/> for the full list.
	Name string `json:"name"`
}

type ActionLocale struct {
	// A localized action name
	Name string `json:"name"`
}

// Dates

func FromDateTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func ToDateTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
