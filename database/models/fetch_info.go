package models

import (
	"strconv"
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/pkg/errors"
)

type FetchInfo struct {
	// Something like "profile", "profile_collections", etc.
	ObjectType string `hades:"primary_key"`
	ObjectID   string `hades:"primary_key"`

	FetchedAt *time.Time
}

func GetFetchInfo(conn *sqlite.Conn, objectType string, objectID int64) (*FetchInfo, error) {
	return GetFetchInfoString(conn, objectType, strconv.FormatInt(objectID, 10))
}

func GetFetchInfoString(conn *sqlite.Conn, objectType string, objectID string) (*FetchInfo, error) {
	var fi FetchInfo
	ok, err := HadesContext().SelectOne(conn, &fi, builder.Eq{
		"object_type": objectType,
		"object_id":   objectID,
	})
	if err != nil {
		return nil, err
	}

	if ok {
		return &fi, nil
	}
	return nil, nil
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

func (ft FetchTarget) Validate() error {
	if ft.Type == "" {
		return errors.Errorf("FetchTarget.Type must be non-empty")
	}
	if ft.StringID == "" && ft.ID == 0 {
		return errors.Errorf("FetchTarget.StringID or FetchTarget.ID must be set")
	}
	if ft.TTL == 0 {
		return errors.Errorf("FetchTarget.TTL must be non-zero")
	}
	return nil
}

func (ft FetchTarget) MustGetInfo(conn *sqlite.Conn) *FetchInfo {
	fi, err := ft.GetInfo(conn)
	Must(err)
	return fi
}

func (ft FetchTarget) GetInfo(conn *sqlite.Conn) (*FetchInfo, error) {
	if ft.StringID != "" {
		return GetFetchInfoString(conn, ft.Type, ft.StringID)
	} else {
		return GetFetchInfo(conn, ft.Type, ft.ID)
	}
}

func (ft FetchTarget) MustIsStale(conn *sqlite.Conn) bool {
	stale, err := ft.IsStale(conn)
	Must(err)
	return stale
}

func (ft FetchTarget) IsStale(conn *sqlite.Conn) (bool, error) {
	err := ft.Validate()
	if err != nil {
		return false, err
	}

	fi, err := ft.GetInfo(conn)
	if err != nil {
		return false, err
	}

	if fi == nil || fi.FetchedAt == nil {
		return true, nil
	}

	if time.Since(*fi.FetchedAt) > ft.TTL {
		return true, nil
	}
	return false, nil
}

func (ft FetchTarget) MarkFresh(conn *sqlite.Conn) error {
	err := ft.Validate()
	if err != nil {
		return err
	}

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

	return HadesContext().Save(conn, fi)
}

func (ft FetchTarget) MustMarkFresh(conn *sqlite.Conn) {
	err := ft.MarkFresh(conn)
	Must(err)
}

func MustMarkAllFresh(conn *sqlite.Conn, fts []FetchTarget) {
	err := MarkAllFresh(conn, fts)
	Must(err)
}

func MarkAllFresh(conn *sqlite.Conn, fts []FetchTarget) error {
	fetchedAt := time.Now().UTC()

	fis := make([]*FetchInfo, len(fts))
	for i, ft := range fts {
		err := ft.Validate()
		if err != nil {
			return err
		}

		fi := &FetchInfo{
			ObjectType: ft.Type,
			FetchedAt:  &fetchedAt,
		}

		if ft.StringID != "" {
			fi.ObjectID = ft.StringID
		} else {
			fi.ObjectID = strconv.FormatInt(ft.ID, 10)
		}
		fis[i] = fi
	}

	return Save(conn, fis)
}
