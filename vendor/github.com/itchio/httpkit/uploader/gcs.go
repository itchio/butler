package uploader

// gcs represents the state of a google cloud
// resumable upload session.
type gcs int

const (
	gcsResume gcs = iota
	gcsNeedQuery
	gcsUploadComplete
	gcsSessionPoisonedOrExpired
	gcsSessionNotFound
	gcsUnknown
)

func (g gcs) String() string {
	switch g {
	case gcsResume:
		return "gcsResume"
	case gcsNeedQuery:
		return "gcsNeedQuery"
	case gcsUploadComplete:
		return "gcsUploadComplete"
	case gcsSessionPoisonedOrExpired:
		return "gcsSessionPoisonedOrExpired"
	case gcsSessionNotFound:
		return "gcsSessionNotFound"
	default:
		return "gcsUnknown"
	}
}

func interpretGcsStatusCode(status int) gcs {
	switch status / 100 {
	case 2:
		if status == 200 || status == 201 {
			return gcsUploadComplete
		}
	case 3:
		if status == 308 {
			return gcsResume
		}
	case 4:
		if status == 410 {
			return gcsSessionPoisonedOrExpired
		} else if status == 404 {
			return gcsSessionNotFound
		} else if status == 408 {
			// sic. not a real 4xx error but a reverse-proxying oddity
			// commit might still have been successful
			return gcsNeedQuery
		}
	case 5:
		// internal server error, bad gateway, service unavailable,
		// gateway timeout, all mean "maybe it did commit, maybe not",
		// need query to find out what was actually committed.
		return gcsNeedQuery
	}

	return gcsUnknown
}
