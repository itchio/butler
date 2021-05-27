package butlerd

import (
	"time"

	"github.com/itchio/hush"
	"github.com/itchio/hush/manifest"

	validation "github.com/go-ozzo/ozzo-validation"
	itchio "github.com/itchio/go-itchio"
)

// When using TCP transport, must be the first message sent
//
// @name Meta.Authenticate
// @category Utilities
// @caller client
type MetaAuthenticateParams struct {
	Secret string `json:"secret"`
}

func (p MetaAuthenticateParams) Validate() error {
	return nil
}

type MetaAuthenticateResult struct {
	OK bool `json:"ok"`
}

// When called, defines the entire duration of the daemon's life.
//
// Cancelling that conversation (or closing the TCP connection) will
// shut down the daemon after all other requests have finished. This
// allows gracefully switching to another daemon.
//
// This conversation is also used to send all global notifications,
// regarding data that's fetched, network state, etc.
//
// Note that this call never returns - you have to cancel it when you're
// done with the daemon.
//
// @name Meta.Flow
// @category Utilities
// @caller client
type MetaFlowParams struct {
}

func (p MetaFlowParams) Validate() error {
	return nil
}

type MetaFlowResult struct {
}

// When called, gracefully shutdown the butler daemon.
// @name Meta.Shutdown
// @category Utilities
// @caller client
type MetaShutdownParams struct {
}

func (p MetaShutdownParams) Validate() error {
	return nil
}

type MetaShutdownResult struct {
}

// The first notification sent when @@MetaFlowParams is called.
//
// @category Utilities
type MetaFlowEstablishedNotification struct {
	// The identifier of the daemon process for which the flow was established
	PID int64 `json:"pid"`
}

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

func (p VersionGetParams) Validate() error {
	return nil
}

// @name Network.SetSimulateOffline
// @category Utilities
// @caller client
type NetworkSetSimulateOfflineParams struct {
	// If true, all operations after this point will behave
	// as if there were no network connections
	Enabled bool `json:"enabled"`
}

func (p NetworkSetSimulateOfflineParams) Validate() error {
	return nil
}

type NetworkSetSimulateOfflineResult struct{}

// @name Network.SetBandwidthThrottle
// @category Utilities
// @caller client
type NetworkSetBandwidthThrottleParams struct {
	// If true, will limit. If false, will clear any bandwidth throttles in place
	Enabled bool `json:"enabled"`
	// The target bandwidth, in kbps
	Rate int64 `json:"rate"`
}

func (p NetworkSetBandwidthThrottleParams) Validate() error {
	return nil
}

type NetworkSetBandwidthThrottleResult struct{}

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

func (p ProfileListParams) Validate() error {
	return nil
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

	// Set to true if you want to force recaptcha
	// @optional
	ForceRecaptcha bool `json:"forceRecaptcha"`
}

func (p ProfileLoginWithPasswordParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Username, validation.Required),
		validation.Field(&p.Password, validation.Required),
	)
}

type ProfileLoginWithPasswordResult struct {
	// Information for the new profile, now remembered
	Profile *Profile `json:"profile"`

	// Profile cookie for website
	Cookie map[string]string `json:"cookie"`
}

// Add a new profile by API key login. This can be used
// for integration tests, for example. Note that no cookies
// are returned for this kind of login.
//
// @name Profile.LoginWithAPIKey
// @category Profile
// @caller client
type ProfileLoginWithAPIKeyParams struct {
	// The API token to use
	APIKey string `json:"apiKey"`
}

func (p ProfileLoginWithAPIKeyParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.APIKey, validation.Required),
	)
}

type ProfileLoginWithAPIKeyResult struct {
	// Information for the new profile, now remembered
	Profile *Profile `json:"profile"`
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

func (p ProfileRequestCaptchaParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.RecaptchaURL, validation.Required),
	)
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

func (p ProfileRequestTOTPParams) Validate() error {
	return nil
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

func (p ProfileUseSavedLoginParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
	)
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

func (p ProfileForgetParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
	)
}

type ProfileForgetResult struct {
	// True if the profile did exist (and was successfully forgotten)
	Success bool `json:"success"`
}

// Stores some data associated to a profile, by key.
//
// @name Profile.Data.Put
// @category Profile
// @caller client
type ProfileDataPutParams struct {
	ProfileID int64  `json:"profileId"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

func (p ProfileDataPutParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
		validation.Field(&p.Key, validation.Required),
		validation.Field(&p.Value, validation.Required),
	)
}

type ProfileDataPutResult struct {
}

// Retrieves some data associated to a profile, by key.
//
// @name Profile.Data.Get
// @category Profile
// @caller client
type ProfileDataGetParams struct {
	ProfileID int64  `json:"profileId"`
	Key       string `json:"key"`
}

func (p ProfileDataGetParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
		validation.Field(&p.Key, validation.Required),
	)
}

type ProfileDataGetResult struct {
	// True if the value existed
	OK    bool   `json:"ok"`
	Value string `json:"value"`
}

//----------------------------------------------------------------------
// Search
//----------------------------------------------------------------------

// Searches for games.
//
// @name Search.Games
// @category Search
// @caller client
type SearchGamesParams struct {
	ProfileID int64 `json:"profileId"`

	Query string `json:"query"`
}

func (p SearchGamesParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
		validation.Field(&p.Query, validation.Required),
	)
}

type SearchGamesResult struct {
	Games []*itchio.Game `json:"games"`
}

// Searches for users.
//
// @name Search.Users
// @category Search
// @caller client
type SearchUsersParams struct {
	ProfileID int64 `json:"profileId"`

	Query string `json:"query"`
}

func (p SearchUsersParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
		validation.Field(&p.Query, validation.Required),
	)
}

type SearchUsersResult struct {
	Users []*itchio.User `json:"users"`
}

//----------------------------------------------------------------------
// Fetch
//----------------------------------------------------------------------

// Fetches information for an itch.io game.
//
// @name Fetch.Game
// @category Fetch
// @caller client
type FetchGameParams struct {
	// Identifier of game to look for
	GameID int64 `json:"gameId"`

	// Force an API request
	// @optional
	Fresh bool `json:"fresh"`
}

func (p FetchGameParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.GameID, validation.Required),
	)
}

func (p FetchGameParams) IsFresh() bool {
	return p.Fresh
}

type FetchGameResult struct {
	// Game info
	Game *itchio.Game `json:"game"`

	// Marks that a request should be issued afterwards with 'Fresh' set
	// @optional
	Stale bool `json:"stale,omitempty"`
}

func (r *FetchGameResult) SetStale(stale bool) {
	r.Stale = stale
}

type GameRecord struct {
	// Game ID
	ID int64 `json:"id"`

	// Game title
	Title string `json:"title"`

	// Game cover
	Cover string `json:"cover,omitempty"`

	// True if owned
	Owned bool `json:"owned,omitempty"`

	// Non-nil if installed (has caves)
	InstalledAt *time.Time `json:"installedAt,omitempty"`
}

// Fetches game records - owned, installed, in collection,
// with search, etc. Includes download key info, cave info, etc.
//
// @name Fetch.GameRecords
// @category Fetch
// @caller client
type FetchGameRecordsParams struct {
	// Profile to use to fetch game
	ProfileID int64 `json:"profileId"`

	// Source from which to fetch games
	Source GameRecordsSource `json:"source"`

	// Collection ID, required if `Source` is "collection"
	// @optional
	CollectionID int64 `json:"collectionId"`

	// Maximum number of games to return at a time
	// @optional
	Limit int64 `json:"limit"`

	// Games to skip
	// @optional
	Offset int64 `json:"offset"`

	// When specified only shows game titles that contain this string
	// @optional
	Search string `json:"search"`

	// Criterion to sort by
	// @optional
	SortBy string `json:"sortBy"`

	// Filters
	// @optional
	Filters GameRecordsFilters `json:"filters"`

	// @optional
	Reverse bool `json:"reverse"`

	// If set, will force fresh data
	// @optional
	Fresh bool `json:"fresh"`
}

type GameRecordsSource string

const (
	// Games for which the profile has a download key
	GameRecordsSourceOwned GameRecordsSource = "owned"
	// Games for which a cave exists (regardless of the profile)
	GameRecordsSourceInstalled GameRecordsSource = "installed"
	// Games authored by profile, or for whom profile is an admin of
	GameRecordsSourceProfile GameRecordsSource = "profile"
	// Games from a collection
	GameRecordsSourceCollection GameRecordsSource = "collection"
)

var GameRecordsSourceList = []interface{}{
	GameRecordsSourceOwned,
	GameRecordsSourceInstalled,
	GameRecordsSourceProfile,
	GameRecordsSourceCollection,
}

type GameRecordsFilters struct {
	// @optional
	Classification itchio.GameClassification `json:"classification"`
	// @optional
	Installed bool `json:"installed"`
	// @optional
	Owned bool `json:"owned"`
}

func (p GameRecordsFilters) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Classification, validation.In(GameClassificationList...)),
	)
}

func (p FetchGameRecordsParams) GetProfileID() int64 {
	return p.ProfileID
}

func (p FetchGameRecordsParams) IsFresh() bool {
	return p.Fresh
}

func (p FetchGameRecordsParams) Validate() error {
	err := validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
		validation.Field(&p.Filters),
		validation.Field(&p.Source, validation.Required, validation.In(GameRecordsSourceList...)),
	)
	if err != nil {
		return err
	}

	switch p.Source {
	case GameRecordsSourceOwned:
		return validation.ValidateStruct(&p,
			validation.Field(&p.SortBy, validation.In("acquiredAt", "title")),
		)
	case GameRecordsSourceProfile:
		return validation.ValidateStruct(&p,
			validation.Field(&p.SortBy, validation.In("title", "views", "downloads", "purchases")),
		)
	case GameRecordsSourceCollection:
		return validation.ValidateStruct(&p,
			validation.Field(&p.CollectionID, validation.Required),
			validation.Field(&p.SortBy, validation.In("default", "title")),
		)
	case GameRecordsSourceInstalled:
		return validation.ValidateStruct(&p,
			validation.Field(&p.SortBy, validation.In("lastTouched", "playTime", "title", "installedSize")),
		)
	}
	return nil
}

type FetchGameRecordsResult struct {
	// All the records that were fetched
	Records []GameRecord `json:"records"`

	// Marks that a request should be issued afterwards with 'Fresh' set
	// @optional
	Stale bool `json:"stale,omitempty"`
}

func (r *FetchGameRecordsResult) SetStale(stale bool) {
	r.Stale = stale
}

// Fetches a download key
//
// @name Fetch.DownloadKey
// @category Fetch
// @caller client
type FetchDownloadKeyParams struct {
	DownloadKeyID int64 `json:"downloadKeyId"`

	ProfileID int64 `json:"profileId"`

	// Force an API request
	// @optional
	Fresh bool `json:"fresh"`
}

func (p FetchDownloadKeyParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.DownloadKeyID, validation.Required),
		validation.Field(&p.ProfileID, validation.Required),
	)
}

func (p FetchDownloadKeyParams) IsFresh() bool {
	return p.Fresh
}

type FetchDownloadKeyResult struct {
	DownloadKey *itchio.DownloadKey `json:"downloadKey"`

	// Marks that a request should be issued afterwards with 'Fresh' set
	// @optional
	Stale bool `json:"stale,omitempty"`
}

// Fetches multiple download keys
//
// @name Fetch.DownloadKeys
// @category Fetch
// @caller client
type FetchDownloadKeysParams struct {
	ProfileID int64 `json:"profileId"`

	// Number of items to skip
	// @optional
	Offset int64 `json:"offset"`

	// Max number of results per page (default = 5)
	// @optional
	Limit int64 `json:"limit"`

	// Filter results
	// @optional
	Filters FetchDownloadKeysFilter `json:"filters"`

	// Force an API request
	// @optional
	Fresh bool `json:"fresh"`
}

type FetchDownloadKeysFilter struct {
	// Return only download keys for given game
	// @optional
	GameID int64 `json:"gameId"`
}

func (p FetchDownloadKeysParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
	)
}

func (p FetchDownloadKeysParams) IsFresh() bool {
	return p.Fresh
}

type FetchDownloadKeysResult struct {
	// All the download keys found in the local DB.
	Items []*itchio.DownloadKey `json:"items"`

	// Whether the information was fetched from a stale cache,
	// and could warrant a refresh if online.
	Stale bool `json:"stale,omitempty"`
}

// Fetches uploads for an itch.io game
//
// @name Fetch.GameUploads
// @category Fetch
// @caller client
type FetchGameUploadsParams struct {
	// Identifier of the game whose uploads we should look for
	GameID int64 `json:"gameId"`

	// Only returns compatible uploads
	OnlyCompatible bool `json:"compatible"`

	// Force an API request
	// @optional
	Fresh bool `json:"fresh"`
}

func (p FetchGameUploadsParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.GameID, validation.Required),
	)
}

func (p FetchGameUploadsParams) IsFresh() bool {
	return p.Fresh
}

type FetchGameUploadsResult struct {
	// List of uploads
	Uploads []*itchio.Upload `json:"uploads"`

	// Marks that a request should be issued
	// afterwards with 'Fresh' set
	// @optional
	Stale bool `json:"stale,omitempty"`
}

func (r *FetchGameUploadsResult) SetStale(stale bool) {
	r.Stale = stale
}

// Fetches builds for an itch.io game
//
// @name Fetch.UploadBuilds
// @category Fetch
// @caller client
type FetchUploadBuildsParams struct {
	// Game whose builds we should look for
	Game *itchio.Game `json:"game"`

	// Upload whose builds we should look for
	Upload *itchio.Upload `json:"upload"`
}

func (p FetchUploadBuildsParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Game, validation.Required),
		validation.Field(&p.Upload, validation.Required),
	)
}

type FetchUploadBuildsResult struct {
	// List of builds
	Builds []*itchio.Build `json:"builds"`
}

// Fetches information for an itch.io user.
//
// @name Fetch.User
// @category Fetch
// @caller client
type FetchUserParams struct {
	// Identifier of the user to look for
	UserID int64 `json:"userId"`

	// Profile to use to look upser
	ProfileID int64 `json:"profileId"`

	// Force an API request
	// @optional
	Fresh bool `json:"fresh"`
}

func (p FetchUserParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.UserID, validation.Required),
		validation.Field(&p.ProfileID, validation.Required),
	)
}

func (p FetchUserParams) IsFresh() bool {
	return p.Fresh
}

type FetchUserResult struct {
	// User info
	User *itchio.User `json:"user"`

	// Marks that a request should be issued
	// afterwards with 'Fresh' set
	// @optional
	Stale bool `json:"stale,omitempty"`
}

func (r *FetchUserResult) SetStale(stale bool) {
	r.Stale = stale
}

// Fetches the best current *locally cached* sale for a given
// game.
//
// @name Fetch.Sale
// @category Fetch
// @caller client
type FetchSaleParams struct {
	// Identifier of the game for which to look for a sale
	GameID int64 `json:"gameId"`
}

func (p FetchSaleParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.GameID, validation.Required),
	)
}

type FetchSaleResult struct {
	// @optional
	Sale *itchio.Sale `json:"sale"`
}

// Fetch a collection's title, gamesCount, etc.
// but not its games.
//
// @name Fetch.Collection
// @category Fetch
// @caller client
type FetchCollectionParams struct {
	// Profile to use to fetch collection
	ProfileID int64 `json:"profileId"`

	// Collection to fetch
	CollectionID int64 `json:"collectionId"`

	// Force an API request before replying.
	// Usually set after getting 'stale' in the response.
	// @optional
	Fresh bool `json:"fresh"`
}

func (p FetchCollectionParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
		validation.Field(&p.CollectionID, validation.Required),
	)
}

func (p FetchCollectionParams) IsFresh() bool {
	return p.Fresh
}

type FetchCollectionResult struct {
	// Collection info
	Collection *itchio.Collection `json:"collection"`

	// True if the info was from local DB and
	// it should be re-queried using "Fresh"
	// @optional
	Stale bool `json:"stale,omitempty"`
}

func (r *FetchCollectionResult) SetStale(stale bool) {
	r.Stale = stale
}

// Fetches information about a collection and the games it
// contains.
//
// @name Fetch.Collection.Games
// @category Fetch
// @caller client
type FetchCollectionGamesParams struct {
	// Profile to use to fetch collection
	ProfileID int64 `json:"profileId"`

	// Identifier of the collection to look for
	CollectionID int64 `json:"collectionId"`

	// Maximum number of games to return at a time.
	// @optional
	Limit int64 `json:"limit"`

	// When specified only shows game titles that contain this string
	// @optional
	Search string `json:"search"`

	// Criterion to sort by
	// @optional
	SortBy string `json:"sortBy"`

	// Filters
	// @optional
	Filters CollectionGamesFilters `json:"filters"`

	// @optional
	Reverse bool `json:"reverse"`

	// Used for pagination, if specified
	// @optional
	Cursor Cursor `json:"cursor"`

	// If set, will force fresh data
	// @optional
	Fresh bool `json:"fresh"`
}

type CollectionGamesFilters struct {
	Installed      bool                      `json:"installed"`
	Classification itchio.GameClassification `json:"classification"`
}

func (p CollectionGamesFilters) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Classification, validation.In(GameClassificationList...)),
	)
}

func (p FetchCollectionGamesParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
		validation.Field(&p.CollectionID, validation.Required),
		validation.Field(&p.Filters),
		validation.Field(&p.SortBy, validation.In("default", "title")),
	)
}

func (p FetchCollectionGamesParams) GetProfileID() int64 {
	return p.ProfileID
}

func (p FetchCollectionGamesParams) GetLimit() int64 {
	return p.Limit
}

func (p FetchCollectionGamesParams) GetCursor() Cursor {
	return p.Cursor
}

func (p FetchCollectionGamesParams) IsFresh() bool {
	return p.Fresh
}

type FetchCollectionGamesResult struct {
	// Requested games for this collection
	Items []*itchio.CollectionGame `json:"items"`

	// Use to fetch the next 'page' of results
	// @optional
	NextCursor Cursor `json:"nextCursor,omitempty"`

	// If true, re-issue request with 'Fresh'
	// @optional
	Stale bool `json:"stale,omitempty"`
}

func (r *FetchCollectionGamesResult) SetStale(stale bool) {
	r.Stale = stale
}

// Lists collections for a profile. Does not contain
// games.
//
// @name Fetch.ProfileCollections
// @category Fetch
// @caller client
type FetchProfileCollectionsParams struct {
	// Profile for which to fetch collections
	ProfileID int64 `json:"profileId"`

	// Maximum number of collections to return at a time.
	// @optional
	Limit int64 `json:"limit"`

	// When specified only shows collection titles that contain this string
	// @optional
	Search string `json:"search"`

	// Criterion to sort by
	// @optional
	SortBy string `json:"sortBy"`

	// @optional
	Reverse bool `json:"reverse"`

	// Used for pagination, if specified
	// @optional
	Cursor Cursor `json:"cursor"`

	// If set, will force fresh data
	// @optional
	Fresh bool `json:"fresh"`
}

func (p FetchProfileCollectionsParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
		validation.Field(&p.SortBy, validation.In("updatedAt", "title")),
	)
}

func (p FetchProfileCollectionsParams) GetCursor() Cursor {
	return p.Cursor
}

func (p FetchProfileCollectionsParams) GetLimit() int64 {
	return p.Limit
}

func (p FetchProfileCollectionsParams) IsFresh() bool {
	return p.Fresh
}

type FetchProfileCollectionsResult struct {
	// Collections belonging to the profile
	Items []*itchio.Collection `json:"items"`

	// Used to fetch the next page
	// @optional
	NextCursor Cursor `json:"nextCursor,omitempty"`

	// If true, re-issue request with "Fresh"
	// @optional
	Stale bool `json:"stale,omitempty"`
}

func (r *FetchProfileCollectionsResult) SetStale(stale bool) {
	r.Stale = stale
}

// @name Fetch.ProfileGames
// @category Fetch
// @caller client
type FetchProfileGamesParams struct {
	// Profile for which to fetch games
	ProfileID int64 `json:"profileId"`

	// Maximum number of items to return at a time.
	// @optional
	Limit int64 `json:"limit"`

	// When specified only shows game titles that contain this string
	// @optional
	Search string `json:"search"`

	// Criterion to sort by
	// @optional
	SortBy string `json:"sortBy"`

	// Filters
	// @optional
	Filters ProfileGameFilters `json:"filters"`

	// @optional
	Reverse bool `json:"reverse"`

	// Used for pagination, if specified
	// @optional
	Cursor Cursor `json:"cursor"`

	// If set, will force fresh data
	// @optional
	Fresh bool `json:"fresh"`
}

type ProfileGameFilters struct {
	Visibility string `json:"visibility"`
	PaidStatus string `json:"paidStatus"`
}

func (p ProfileGameFilters) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Visibility, validation.In("draft", "published")),
		validation.Field(&p.PaidStatus, validation.In("paid", "free")),
	)
}

func (p FetchProfileGamesParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
		validation.Field(&p.SortBy, validation.In("default", "title", "views", "downloads", "purchases")),
		validation.Field(&p.Filters),
	)
}

func (p FetchProfileGamesParams) GetProfileID() int64 {
	return p.ProfileID
}

func (p FetchProfileGamesParams) GetCursor() Cursor {
	return p.Cursor
}

func (p FetchProfileGamesParams) GetLimit() int64 {
	return p.Limit
}

func (p FetchProfileGamesParams) IsFresh() bool {
	return p.Fresh
}

type ProfileGame struct {
	Game *itchio.Game `json:"game"`

	ViewsCount     int64 `json:"viewsCount"`
	DownloadsCount int64 `json:"downloadsCount"`
	PurchasesCount int64 `json:"purchasesCount"`

	Published bool `json:"published"`
}

type FetchProfileGamesResult struct {
	// Profile games
	Items []*ProfileGame `json:"items"`

	// Used to fetch the next page
	// @optional
	NextCursor Cursor `json:"nextCursor,omitempty"`

	// If true, re-issue request with "Fresh"
	// @optional
	Stale bool `json:"stale,omitempty"`
}

func (r *FetchProfileGamesResult) SetStale(stale bool) {
	r.Stale = stale
}

// @name Fetch.ProfileOwnedKeys
// @category Fetch
// @caller client
type FetchProfileOwnedKeysParams struct {
	// Profile to use to fetch game
	ProfileID int64 `json:"profileId"`

	// Maximum number of owned keys to return at a time.
	// @optional
	Limit int64 `json:"limit"`

	// When specified only shows game titles that contain this string
	// @optional
	Search string `json:"search"`

	// Criterion to sort by
	// @optional
	SortBy string `json:"sortBy"`

	// Filters
	// @optional
	Filters ProfileOwnedKeysFilters `json:"filters"`

	// @optional
	Reverse bool `json:"reverse"`

	// Used for pagination, if specified
	// @optional
	Cursor Cursor `json:"cursor"`

	// If set, will force fresh data
	// @optional
	Fresh bool `json:"fresh"`
}

type ProfileOwnedKeysFilters struct {
	Installed      bool                      `json:"installed"`
	Classification itchio.GameClassification `json:"classification"`
}

func (p ProfileOwnedKeysFilters) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Classification, validation.In(GameClassificationList...)),
	)
}

func (p FetchProfileOwnedKeysParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ProfileID, validation.Required),
		validation.Field(&p.Filters),
		validation.Field(&p.SortBy, validation.In("acquiredAt", "title")),
	)
}

func (p FetchProfileOwnedKeysParams) GetProfileID() int64 {
	return p.ProfileID
}

func (p FetchProfileOwnedKeysParams) IsFresh() bool {
	return p.Fresh
}

func (p FetchProfileOwnedKeysParams) GetCursor() Cursor {
	return p.Cursor
}

func (p FetchProfileOwnedKeysParams) GetLimit() int64 {
	return p.Limit
}

type FetchProfileOwnedKeysResult struct {
	// Download keys fetched for profile
	Items []*itchio.DownloadKey `json:"items"`

	// Used to fetch the next page
	// @optional
	NextCursor Cursor `json:"nextCursor,omitempty"`

	// If true, re-issue request with "Fresh"
	// @optional
	Stale bool `json:"stale,omitempty"`
}

func (r *FetchProfileOwnedKeysResult) SetStale(stale bool) {
	r.Stale = stale
}

// @name Fetch.Commons
// @category Fetch
// @caller client
type FetchCommonsParams struct{}

func (p FetchCommonsParams) Validate() error {
	return nil
}

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
	CreatedAt *time.Time `json:"createdAt"`
}

type CaveSummary struct {
	ID string `json:"id"`

	GameID int64 `json:"gameId"`

	LastTouchedAt *time.Time `json:"lastTouchedAt"`
	SecondsRun    int64      `json:"secondsRun"`
	InstalledSize int64      `json:"installedSize"`
}

// A Cave corresponds to an "installed item" for a game.
//
// It maps one-to-one with an upload. There might be 0, 1, or several
// caves for a given game. Multiple caves for a single game is a rare-ish
// case (single-page bundles, bonus content) but one that should be handled.
type Cave struct {
	// Unique identifier of this cave (UUID)
	ID string `json:"id"`

	// Game that's installed in this cave
	Game *itchio.Game `json:"game"`
	// Upload that's installed in this cave
	Upload *itchio.Upload `json:"upload"`
	// Build that's installed in this cave, if the upload is wharf-powered
	// @optional
	Build *itchio.Build `json:"build"`

	// Stats about cave usage and first install
	Stats *CaveStats `json:"stats"`
	// Information about where the cave is installed, how much space it takes up etc.
	InstallInfo *CaveInstallInfo `json:"installInfo"`
}

// CaveStats contains stats about cave usage and first install
type CaveStats struct {
	// Time the cave was first installed
	InstalledAt   *time.Time `json:"installedAt"`
	LastTouchedAt *time.Time `json:"lastTouchedAt"`
	SecondsRun    int64      `json:"secondsRun"`
}

// CaveInstallInfo contains information about where the cave is installed, how
// much space it takes up, etc.
type CaveInstallInfo struct {
	// Size the cave takes up - or at least, size it took up when we finished
	// installing it. Does not include files generated by the game in the install folder.
	InstalledSize int64 `json:"installedSize"`
	// Name of the install location for this cave. This may change if the cave
	// is moved.
	InstallLocation string `json:"installLocation"`
	// Absolute path to the install folder
	InstallFolder string `json:"installFolder"`
	// If true, this cave is ignored while checking for updates
	Pinned bool `json:"pinned,omitempty"`
}

type InstallLocationSummary struct {
	// Unique identifier for this install location
	ID string `json:"id"`
	// Absolute path on disk for this install location
	Path string `json:"path"`
	// Information about the size used and available at this install location
	SizeInfo *InstallLocationSizeInfo `json:"sizeInfo,omitempty"`
}

type InstallLocationSizeInfo struct {
	// Number of bytes used by caves installed in this location
	InstalledSize int64 `json:"installedSize"`
	// Free space at this location (depends on the partition/disk on which
	// it is), or a negative value if we can't find it
	FreeSize int64 `json:"freeSize"`
	// Total space of this location (depends on the partition/disk on which
	// it is), or a negative value if we can't find it
	TotalSize int64 `json:"totalSize"`
}

// Retrieve info for all caves.
//
// @name Fetch.Caves
// @category Fetch
// @caller client
type FetchCavesParams struct {
	// Maximum number of caves to return at a time.
	// @optional
	Limit int64 `json:"limit"`

	// When specified only shows game titles that contain this string
	// @optional
	Search string `json:"search"`

	// @optional
	SortBy string `json:"sortBy"`

	// Filters
	// @optional
	Filters CavesFilters `json:"filters"`

	// @optional
	Reverse bool `json:"reverse"`

	// Used for pagination, if specified
	// @optional
	Cursor Cursor `json:"cursor"`
}

type CavesFilters struct {
	// @optional
	Classification itchio.GameClassification `json:"classification"`

	// @optional
	GameID int64 `json:"gameId"`

	// @optional
	InstallLocationID string `json:"installLocationId"`
}

func (p CavesFilters) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Classification, validation.In(GameClassificationList...)),
	)
}

func (p FetchCavesParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Filters),
		validation.Field(&p.SortBy, validation.In("lastTouched", "playTime", "title", "installedSize", "installedAt")),
	)
}

func (p FetchCavesParams) GetLimit() int64 {
	return p.Limit
}

func (p FetchCavesParams) GetCursor() Cursor {
	return p.Cursor
}

type FetchCavesResult struct {
	Items []*Cave `json:"items"`

	// Use to fetch the next 'page' of results
	// @optional
	NextCursor Cursor `json:"nextCursor,omitempty"`
}

// Retrieve info on a cave by ID.
//
// @name Fetch.Cave
// @category Fetch
// @caller client
type FetchCaveParams struct {
	CaveID string `json:"caveId"`
}

func (p FetchCaveParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.CaveID, validation.Required),
	)
}

type FetchCaveResult struct {
	Cave *Cave `json:"cave"`
}

// Mark all local data as stale.
//
// @name Fetch.ExpireAll
// @category Fetch
// @caller client
type FetchExpireAllParams struct{}

func (p FetchExpireAllParams) Validate() error {
	return nil
}

type FetchExpireAllResult struct{}

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

func (p GameFindUploadsParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Game, validation.Required),
	)
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

	// Don't run install prepare (assume we can just run it at perform time)
	// @optional
	FastQueue bool `json:"fastQueue"`
}

func (p InstallQueueParams) Validate() error {
	return nil
}

type InstallQueueResult struct {
	ID                string         `json:"id"`
	Reason            DownloadReason `json:"reason"`
	CaveID            string         `json:"caveId"`
	Game              *itchio.Game   `json:"game"`
	Upload            *itchio.Upload `json:"upload"`
	Build             *itchio.Build  `json:"build"`
	InstallFolder     string         `json:"installFolder"`
	StagingFolder     string         `json:"stagingFolder"`
	InstallLocationID string         `json:"installLocationId"`
}

// For modal-first install
//
// @name Install.Plan
// @category Install
// @caller client
type InstallPlanParams struct {
	// The ID of the game we're planning to install
	GameID int64 `json:"gameId"`

	// The download session ID to use for this install plan
	// @optional
	DownloadSessionID string `json:"downloadSessionId"`

	// @optional
	UploadID int64 `json:"uploadId"`
}

func (p InstallPlanParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.GameID, validation.Required),
	)
}

type InstallPlanResult struct {
	Game    *itchio.Game     `json:"game"`
	Uploads []*itchio.Upload `json:"uploads"`

	Info *InstallPlanInfo `json:"info"`
}

type InstallPlanInfo struct {
	Upload    *itchio.Upload `json:"upload"`
	Build     *itchio.Build  `json:"build"`
	Type      string         `json:"type"`
	DiskUsage *DiskUsageInfo `json:"diskUsage"`

	Error        string `json:"error,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	ErrorCode    int64  `json:"errorCode,omitempty"`
}

type DiskUsageInfo struct {
	FinalDiskUsage  int64  `json:"finalDiskUsage"`
	NeededFreeSpace int64  `json:"neededFreeSpace"`
	Accuracy        string `json:"accuracy"`
}

// @name Caves.SetPinned
// @category Install
// @caller client
type CavesSetPinnedParams struct {
	// ID of the cave to pin/unpin
	CaveID string `json:"caveId"`

	// Pinned state the cave should have after this call
	Pinned bool `json:"pinned"`
}

func (p CavesSetPinnedParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.CaveID, validation.Required),
	)
}

type CavesSetPinnedResult struct{}

// Create a shortcut for an existing cave .
//
// @name Install.CreateShortcut
// @category Install
// @caller client
type InstallCreateShortcutParams struct {
	CaveID string `json:"caveId"`
}

func (p InstallCreateShortcutParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.CaveID, validation.Required),
	)
}

type InstallCreateShortcutResult struct {
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

func (p InstallPerformParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ID, validation.Required),
		validation.Field(&p.StagingFolder, validation.Required),
	)
}

type InstallPerformResult struct {
	CaveID string              `json:"caveId"`
	Events []hush.InstallEvent `json:"events"`
}

// Attempt to gracefully cancel an ongoing operation.
//
// @name Install.Cancel
// @category Install
// @caller client
type InstallCancelParams struct {
	// The UUID of the task to cancel, as passed to @@OperationStartParams
	ID string `json:"id"`
}

func (p InstallCancelParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ID, validation.Required),
	)
}

type InstallCancelResult struct {
	DidCancel bool `json:"didCancel"`
}

// UninstallParams contains all the parameters needed to perform
// an uninstallation for a game via @@OperationStartParams.
//
// @name Uninstall.Perform
// @category Install
// @caller client
type UninstallPerformParams struct {
	// The cave to uninstall
	CaveID string `json:"caveId"`

	// If true, don't attempt to run any uninstallers, just
	// remove the DB record and burn the install folder to the ground.
	// @optional
	Hard bool `json:"hard"`
}

func (p UninstallPerformParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.CaveID, validation.Required),
	)
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

func (p InstallVersionSwitchQueueParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.CaveID, validation.Required),
	)
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

func (p InstallVersionSwitchPickParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Cave, validation.Required),
		validation.Field(&p.Upload, validation.Required),
	)
}

type InstallVersionSwitchPickResult struct {
	// A negative index aborts the version switch
	Index int64 `json:"index"`
}

// GameCredentials contains all the credentials required to make API requests
// including the download key if any.
type GameCredentials struct {
	// A valid itch.io API key
	APIKey string `json:"apiKey"`
	// A download key identifier, or 0 if no download key is available
	// @optional
	DownloadKey int64 `json:"downloadKey"`
}

func (gc *GameCredentials) JustAPIKey() *GameCredentials {
	return &GameCredentials{
		APIKey: gc.APIKey,
	}
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

func (p PickUploadParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Uploads, validation.Required),
	)
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

// @name Install.Locations.List
// @category Install
// @caller client
type InstallLocationsListParams struct {
}

func (p InstallLocationsListParams) Validate() error {
	return nil
}

type InstallLocationsListResult struct {
	InstallLocations []*InstallLocationSummary `json:"installLocations"`
}

// @name Install.Locations.Add
// @category Install
// @caller client
type InstallLocationsAddParams struct {
	// identifier of the new install location.
	// if not specified, will be generated.
	// @optional
	ID string `json:"id"`

	// path of the new install location
	Path string `json:"path"`
}

func (p InstallLocationsAddParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Path, validation.Required),
	)
}

type InstallLocationsAddResult struct {
	InstallLocation *InstallLocationSummary `json:"installLocation"`
}

// @name Install.Locations.Remove
// @category Install
// @caller client
type InstallLocationsRemoveParams struct {
	// identifier of the install location to remove
	ID string `json:"id"`
}

func (p InstallLocationsRemoveParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ID, validation.Required),
	)
}

type InstallLocationsRemoveResult struct {
}

// @name Install.Locations.GetByID
// @category Install
// @caller client
type InstallLocationsGetByIDParams struct {
	// identifier of the install location to remove
	ID string `json:"id"`
}

func (p InstallLocationsGetByIDParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ID, validation.Required),
	)
}

type InstallLocationsGetByIDResult struct {
	InstallLocation *InstallLocationSummary `json:"installLocation"`
}

// @name Install.Locations.Scan
// @category Install
// @caller client
type InstallLocationsScanParams struct {
	// path to a legacy marketDB
	// @optional
	LegacyMarketPath string `json:"legacyMarketPath"`
}

func (p InstallLocationsScanParams) Validate() error {
	return nil
}

// Sent during @@InstallLocationsScanParams whenever
// a game is found.
//
// @name Install.Locations.Scan.Yield
// @category Install
type InstallLocationsScanYieldNotification struct {
	Game *itchio.Game `json:"game"`
}

// Sent at the end of @@InstallLocationsScanParams
//
// @name Install.Locations.Scan.ConfirmImport
// @category Install
// @caller server
type InstallLocationsScanConfirmImportParams struct {
	// number of items that will be imported
	NumItems int64 `json:"numItems"`
}

func (p InstallLocationsScanConfirmImportParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.NumItems, validation.Required),
	)
}

type InstallLocationsScanConfirmImportResult struct {
	Confirm bool `json:"confirm"`
}

type InstallLocationsScanResult struct {
	NumFoundItems    int64 `json:"numFoundItems"`
	NumImportedItems int64 `json:"numImportedItems"`
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

func (p DownloadsQueueParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Item, validation.Required),
	)
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

func (p DownloadsPrioritizeParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.DownloadID, validation.Required),
	)
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

func (p DownloadsListParams) Validate() error {
	return nil
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

func (p DownloadsClearFinishedParams) Validate() error {
	return nil
}

type DownloadsClearFinishedResult struct {
}

// Drive downloads, which is: perform them one at a time,
// until they're all finished.
//
// @name Downloads.Drive
// @category Downloads
// @caller client
type DownloadsDriveParams struct{}

func (p DownloadsDriveParams) Validate() error {
	return nil
}

type DownloadsDriveResult struct{}

// Stop driving downloads gracefully.
//
// @name Downloads.Drive.Cancel
// @category Downloads
// @caller client
type DownloadsDriveCancelParams struct{}

func (p DownloadsDriveCancelParams) Validate() error {
	return nil
}

type DownloadsDriveCancelResult struct {
	DidCancel bool `json:"didCancel"`
}

// @name Downloads.Drive.Progress
type DownloadsDriveProgressNotification struct {
	Download *Download         `json:"download"`
	Progress *DownloadProgress `json:"progress"`
	// BPS values for the last minute
	SpeedHistory []float64 `json:"speedHistory"`
}

// @name Downloads.Drive.Started
type DownloadsDriveStartedNotification struct {
	Download *Download `json:"download"`
}

// @name Downloads.Drive.Errored
type DownloadsDriveErroredNotification struct {
	// The download that errored. It contains all the error
	// information: a short message, a full stack trace,
	// and a butlerd error code.
	Download *Download `json:"download"`
}

// @name Downloads.Drive.Finished
type DownloadsDriveFinishedNotification struct {
	Download *Download `json:"download"`
}

// @name Downloads.Drive.Discarded
type DownloadsDriveDiscardedNotification struct {
	Download *Download `json:"download"`
}

// Sent during @@DownloadsDriveParams to inform on network
// status changes.
//
// @name Downloads.Drive.NetworkStatus
type DownloadsDriveNetworkStatusNotification struct {
	// The current network status
	Status NetworkStatus `json:"status"`
}

type NetworkStatus string

const (
	NetworkStatusOnline  NetworkStatus = "online"
	NetworkStatusOffline NetworkStatus = "offline"
)

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
	ErrorMessage  *string        `json:"errorMessage"`
	ErrorCode     *int64         `json:"errorCode"`
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

func (p DownloadsRetryParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.DownloadID, validation.Required),
	)
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

func (p DownloadsDiscardParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.DownloadID, validation.Required),
	)
}

type DownloadsDiscardResult struct{}

//----------------------------------------------------------------------
// CheckUpdate
//----------------------------------------------------------------------

// Looks for game updates.
//
// If a list of cave identifiers is passed, will only look for
// updates for these caves *and will ignore snooze*.
//
// Otherwise, will look for updates for all games, respecting snooze.
//
// Updates found are regularly sent via @@GameUpdateAvailableNotification, and
// then all at once in the result.
//
// @category Update
// @caller client
type CheckUpdateParams struct {
	// If specified, will only look for updates to these caves
	// @optional
	CaveIDs []string `json:"caveIds"`

	// If specified, will log information even when we have no warnings/errors
	// @optional
	Verbose bool `json:"verbose"`
}

func (p CheckUpdateParams) Validate() error {
	return nil
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
	// Cave we found an update for
	CaveID string `json:"caveId"`

	// Game we found an update for
	Game *itchio.Game `json:"game"`

	// True if this is a direct update, ie. we're on
	// a channel that still exists, and there's a new build
	// False if it's an indirect update, for example a new
	// upload that appeared after we installed, but we're
	// not sure if it's an upgrade or other additional content
	Direct bool `json:"direct"`

	// Available choice of updates
	Choices []*GameUpdateChoice `json:"choices"`
}

// One possible upload/build choice to upgrade a cave
//
// @category update
type GameUpdateChoice struct {
	// Upload to be installed
	Upload *itchio.Upload `json:"upload"`
	// Build to be installed (may be nil)
	Build *itchio.Build `json:"build"`
	// How confident we are that this is the right upgrade
	Confidence float64 `json:"confidence"`
}

// Snoozing a cave means we ignore all new uploads (that would
// be potential updates) between the cave's last install operation
// and now.
//
// This can be undone by calling @@CheckUpdateParams with this specific
// cave identifier.
//
// @category Update
// @caller client
type SnoozeCaveParams struct {
	CaveID string `json:"caveId"`
}

func (p SnoozeCaveParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.CaveID, validation.Required),
	)
}

type SnoozeCaveResult struct {
}

//----------------------------------------------------------------------
// Launch
//----------------------------------------------------------------------

// Attempt to launch an installed game.
//
// @name Launch
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
	// @optional
	Sandbox bool `json:"sandbox,omitempty"`
}

func (p LaunchParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.CaveID, validation.Required),
		validation.Field(&p.PrereqsDir, validation.Required),
	)
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

// Sent during @@LaunchParams if the game/application comes with a service license
// agreement.
//
// @tags Dialogs
// @category Launch
// @caller server
type AcceptLicenseParams struct {
	// The full text of the license agreement, in its default
	// language, which is usually English.
	Text string `json:"text"`
}

func (p AcceptLicenseParams) Validate() error {
	return nil
}

type AcceptLicenseResult struct {
	// true if the user accepts the terms of the license, false otherwise.
	// Note that false will cancel the launch.
	Accept bool `json:"accept"`
}

// Sent during @@LaunchParams, ask the user to pick a manifest action to launch.
//
// See [itch app manifests](https://itch.io/docs/itch/integrating/manifest.html).
//
// @tags Dialogs
// @category Launch
// @caller server
type PickManifestActionParams struct {
	// A list of actions to pick from. Must be shown to the user in the order they're passed.
	Actions []*manifest.Action `json:"actions"`
}

func (p PickManifestActionParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Actions, validation.Required),
	)
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

func (p ShellLaunchParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ItemPath, validation.Required),
	)
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

func (p HTMLLaunchParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.RootFolder, validation.Required),
		validation.Field(&p.IndexPath, validation.Required),
	)
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

func (p URLLaunchParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.URL, validation.Required),
	)
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

func (p AllowSandboxSetupParams) Validate() error {
	return nil
}

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

func (p PrereqsFailedParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Error, validation.Required),
	)
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

func (p CleanDownloadsSearchParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Roots, validation.Required),
	)
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

func (p CleanDownloadsApplyParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Entries, validation.Required),
	)
}

// @category Clean Downloads
type CleanDownloadsApplyResult struct{}

//----------------------------------------------------------------------
// System
//----------------------------------------------------------------------

// Get information on a filesystem.
//
// @name System.StatFS
// @category System
// @caller client
type SystemStatFSParams struct {
	Path string `json:"path"`
}

func (p SystemStatFSParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Path, validation.Required),
	)
}

type SystemStatFSResult struct {
	FreeSize  int64 `json:"freeSize"`
	TotalSize int64 `json:"totalSize"`
}

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

func (p TestDoubleTwiceParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Number, validation.Required),
	)
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

func (p TestDoubleParams) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Number, validation.Required),
	)
}

// Result for Test.Double
type TestDoubleResult struct {
	// The number, doubled
	Number int64 `json:"number"`
}

// butlerd JSON-RPC 2.0 error codes
type Code int64

// Note: codes -32000 to -32099 are reserved for the JSON-RPC
// protocol.

const (
	// An operation was cancelled gracefully
	CodeOperationCancelled Code = 499
	// An operation was aborted by the user
	CodeOperationAborted Code = 410

	// We tried to launch something, but the install folder just wasn't there
	CodeInstallFolderDisappeared Code = 404

	// We tried to install something, but could not find compatible uploads
	CodeNoCompatibleUploads Code = 2001

	// This title is hosted on an incompatible third-party website
	CodeUnsupportedHost Code = 3001

	// Nothing that can be launched was found
	CodeNoLaunchCandidates Code = 5000

	// Java Runtime Environment is required to launch this title.
	CodeJavaRuntimeNeeded Code = 6000

	// There is no Internet connection
	CodeNetworkDisconnected Code = 9000

	// API error
	CodeAPIError Code = 12000

	// The database is busy
	CodeDatabaseBusy Code = 16000

	// An install location could not be removed because it has active downloads
	CodeCantRemoveLocationBecauseOfActiveDownloads Code = 18000
)

// Dates

func FromDateTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func ToDateTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// Cursors

type Cursor string

// Validation enums

var GameClassificationList = []interface{}{
	itchio.GameClassificationGame,
	itchio.GameClassificationTool,
	itchio.GameClassificationAssets,
	itchio.GameClassificationGameMod,
	itchio.GameClassificationPhysicalGame,
	itchio.GameClassificationSoundtrack,
	itchio.GameClassificationOther,
	itchio.GameClassificationComic,
	itchio.GameClassificationBook,
}
