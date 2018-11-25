package mitch

import (
	"sync"
	"time"
)

type Store struct {
	Users      map[int64]*User
	APIKeys    map[int64]*APIKey
	Games      map[int64]*Game
	Uploads    map[int64]*Upload
	Builds     map[int64]*Build
	BuildFiles map[int64]*BuildFile
	GameAdmins map[int64]*GameAdmin

	CDNFiles map[string]*CDNFile

	idSeed     int64
	writeMutex sync.Mutex
}

func newStore() *Store {
	return &Store{
		Users:      make(map[int64]*User),
		APIKeys:    make(map[int64]*APIKey),
		Games:      make(map[int64]*Game),
		Uploads:    make(map[int64]*Upload),
		Builds:     make(map[int64]*Build),
		BuildFiles: make(map[int64]*BuildFile),
		GameAdmins: make(map[int64]*GameAdmin),

		CDNFiles: make(map[string]*CDNFile),
		idSeed:   10,
	}
}

type User struct {
	Store *Store

	ID             int64
	Username       string
	DisplayName    string
	Gamer          bool
	Developer      bool
	PressUser      bool
	AllowTelemetry bool
}

type APIKey struct {
	Store *Store

	ID        int64
	UserID    int64
	Key       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Game struct {
	Store *Store

	Type           string
	Classification string
	ID             int64
	UserID         int64
	Title          string
	MinPrice       int64
	Published      bool
}

type Upload struct {
	Store *Store

	ID          int64
	GameID      int64
	Filename    string
	URL         string
	Size        int64
	ChannelName string
	Storage     string
	Head        int64
	Type        string

	PlatformWindows bool
	PlatformLinux   bool
	PlatformMac     bool
}

type Build struct {
	Store *Store

	ID            int64
	ParentBuildID int64
	UploadID      int64
	Version       int
}

type BuildFile struct {
	Store *Store

	ID      int64
	BuildID int64
	Type    string
	SubType string
	Status  string

	Filename string
	Size     int64
}

type GameAdmin struct {
	Store *Store

	ID     int64
	GameID int64
	UserID int64
}

type CDNFile struct {
	Path     string
	Filename string
	Size     int64
	Contents []byte
}
