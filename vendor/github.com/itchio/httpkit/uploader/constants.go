package uploader

import "time"

const gcsChunkSize = 256 * 1024 // 256KB

var resumableMaxRetries = fromEnv("WHARF_MAX_RETRIES", 15)
var resumableConnectTimeout = time.Duration(fromEnv("WHARF_CONNECT_TIMEOUT", 30)) * time.Second
var resumableIdleTimeout = time.Duration(fromEnv("WHARF_IDLE_TIMEOUT", 60)) * time.Second
var resumableVerboseDebug = fromEnv("WHARF_VERBOSE_DEBUG", 0)
