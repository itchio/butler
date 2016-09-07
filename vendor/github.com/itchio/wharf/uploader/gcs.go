//go:generate stringer -type=gcsStatus
package uploader

type gcsStatus int

const (
	GcsResume gcsStatus = iota
	GcsNeedQuery
	GcsUploadComplete
	GcsSessionPoisonedOrExpired
	GcsSessionNotFound
	GcsUnknown
)

func interpretGcsStatusCode(status int) gcsStatus {
	switch status / 100 {
	case 2:
		if status == 200 || status == 201 {
			return GcsUploadComplete
		}
	case 3:
		if status == 308 {
			return GcsResume
		}
	case 4:
		if status == 410 {
			return GcsSessionPoisonedOrExpired
		} else if status == 404 {
			return GcsSessionNotFound
		} else if status == 408 {
			// sic. not a real 4xx error but a reverse-proxying oddity
			// commit might still have been successful
			return GcsNeedQuery
		}
	case 5:
		// internal server error, bad gateway, service unavailable,
		// gateway timeout, all mean "maybe it did commit, maybe not",
		// need query to find out what was actually committed.
		return GcsNeedQuery
	}

	return GcsUnknown
}
