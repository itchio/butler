package sz

import "sync"

//==========================
// OutStream
//==========================

var outStreams = make(map[int64]*OutStream)
var outStreamMutex sync.Mutex
var outStreamSeed int64 = 1000

func reserveOutStreamId(out *OutStream) {
	outStreamMutex.Lock()
	defer outStreamMutex.Unlock()
	out.id = outStreamSeed
	outStreamSeed += 1
	outStreams[out.id] = out
}

func freeOutStreamId(id int64) {
	outStreamMutex.Lock()
	defer outStreamMutex.Unlock()
	delete(outStreams, id)
}

//==========================
// InStream
//==========================

var inStreams = make(map[int64]*InStream)
var inStreamSeed int64 = 2000
var inStreamMutex sync.Mutex

func reserveInStreamId(in *InStream) {
	inStreamMutex.Lock()
	defer inStreamMutex.Unlock()
	in.id = inStreamSeed
	inStreamSeed += 1
	inStreams[in.id] = in
}

func freeInStreamId(id int64) {
	inStreamMutex.Lock()
	defer inStreamMutex.Unlock()
	delete(inStreams, id)
}

//==========================
// ExtractCallback
//==========================

var extractCallbacks = make(map[int64]*ExtractCallback)
var extractCallbackSeed int64 = 3000
var extractCallbackMutex sync.Mutex

func reserveExtractCallbackId(ec *ExtractCallback) {
	extractCallbackMutex.Lock()
	defer extractCallbackMutex.Unlock()
	ec.id = extractCallbackSeed
	extractCallbackSeed += 1
	extractCallbacks[ec.id] = ec
}

func freeExtractCallbackId(id int64) {
	extractCallbackMutex.Lock()
	defer extractCallbackMutex.Unlock()
	delete(extractCallbacks, id)
}
