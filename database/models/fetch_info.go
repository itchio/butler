package models

import (
	"strconv"
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
)

type FetchInfo struct {
	// Something like "profile", "profile_collections", etc.
	ObjectType string `hades:"primary_key"`
	ObjectID   string `hades:"primary_key"`

	FetchedAt *time.Time
}

func GetFetchInfo(conn *sqlite.Conn, objectType string, objectID int64) *FetchInfo {
	return GetFetchInfoString(conn, objectType, strconv.FormatInt(objectID, 10))
}

func GetFetchInfoString(conn *sqlite.Conn, objectType string, objectID string) *FetchInfo {
	var fi FetchInfo
	if MustSelectOne(conn, &fi, builder.Eq{
		"object_type": objectType,
		"object_id":   objectID,
	}) {
		return &fi
	}
	return nil
}

type FetchTarget struct {
	// snake_case, like "profile_collections", "collection", "collection_games", etc.
	Type string

	// if non-empty, will be used
	StringID string

	// if StringID is empty, this is used
	ID int64

	// age after which a resource is considered stale
	TTL time.Duration
}

func (ft FetchTarget) Validate() {
	if ft.Type == "" {
		panic("FetchTarget.Type must be non-empty")
	}
	if ft.StringID == "" && ft.ID == 0 {
		panic("FetchTarget.StringID or FetchTarget.ID must be set")
	}
	if ft.TTL == 0 {
		panic("FetchTarget.TTL must be non-zero")
	}
}

func (ft FetchTarget) Info(conn *sqlite.Conn) *FetchInfo {
	if ft.StringID != "" {
		return GetFetchInfoString(conn, ft.Type, ft.StringID)
	} else {
		return GetFetchInfo(conn, ft.Type, ft.ID)
	}
}

func (ft FetchTarget) IsStale(conn *sqlite.Conn) bool {
	ft.Validate()
	fi := ft.Info(conn)
	if fi == nil || fi.FetchedAt == nil {
		return true
	}

	if time.Since(*fi.FetchedAt) > ft.TTL {
		return true
	}
	return false
}

func (ft FetchTarget) MarkFresh(conn *sqlite.Conn) {
	ft.Validate()
	fetchedAt := time.Now().UTC()
	fi := &FetchInfo{
		ObjectType: ft.Type,
		FetchedAt:  &fetchedAt,
	}

	if ft.StringID != "" {
		fi.ObjectID = ft.StringID
	} else {
		fi.ObjectID = strconv.FormatInt(ft.ID, 10)
	}

	MustSave(conn, fi)
}
