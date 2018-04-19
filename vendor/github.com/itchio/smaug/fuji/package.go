// Package fuji implements a sandbox for Windows. It works by
// creating a less-privileged user, `itch-player-XXXXX`, which
// we hide from login and share a game's folder before we launch
// it (then unshare it immediately after).
//
// If you want to see/manage the user the sandbox creates,
// you can use "lusrmgr.msc" on Windows (works in Win+R)
package fuji
