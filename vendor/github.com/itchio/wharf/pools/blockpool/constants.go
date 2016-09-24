package blockpool

// BigBlockSize is the size of blocks stored and fetched by blockpool's sources and sinks
const BigBlockSize int64 = 4 * 1024 * 1024 // 4MB

// ZstdLevel is the quality used when compressing blocks with zstd
const ZstdLevel = 9
