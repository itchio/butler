package operate

import "strings"

// We can't download silently from any of these hosts.
// This maps host names to fun trivia.
var BadExternalHosts = map[string]string{
	// Dropbox-likes (cloud/sync storage)
	"dropbox.com":       "Needs a browser",
	"drive.google.com":  "Needs, like, a mountain of javascript to be interpreted first.",
	"docs.google.com":   "See drive.google.com",
	"goo.gl":            "Nothing good can hide behind that URL shortener (probably drive)",
	"onedrive.live.com": "Microsoft's twist on Dropbox",
	"1drv.ms":           "Short url for OneDrive",
	"yadi.sk":           "Yandex dropbox-like?",

	// Megaupload-likes (classic storage)
	"mega.nz":       "Decrypts files in your browser? What could go wrong.",
	"mega.co.nz":    "Old domain name for mega.nz",
	"mediafire.com": "One of the least terrible 'hosting sites' of its genre, but still no dice.",
	"indiedb.com":   "Also needs a browser",

	// We-transfer likes
	"wetransfer.com": "Needs a browser three",
	"hightail.com":   "Looks like a WeTransfer clone?",

	// Just store pages
	"play.google.com":          "Even if we could download from there, what are we going to do with an APK",
	"itunes.apple.com":         "Ditto",
	"steampowered.com":         "That's not a CDN, just a store page. Don't set your upload to that..",
	"support.steampowered.com": "Why",
	"store.steampowered.com":   "What",
	"gamejolt.com":             "Please stop",

	// HTML5 embed hosts
	"puzzlescript.net": "Great little engine, nothing for us to download though.",
	"philome.la":       "Twine embeds, no archive to download from there",
	"scratch.mit.edu":  "More embeds",

	// ???
	"youtube.com": "Why??",
}

func IsBadExternalHost(host string) bool {
	host = strings.TrimPrefix(host, "www.")
	_, bad := BadExternalHosts[host]
	return bad
}
