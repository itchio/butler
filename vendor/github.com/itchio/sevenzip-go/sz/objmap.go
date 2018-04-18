package sz

import (
	"sync"
	"sync/atomic"
)

var seed int64 = 1

//==========================
// OutStream
//==========================

var outStreams sync.Map

func reserveOutStreamId(obj *OutStream) {
	obj.id = atomic.AddInt64(&seed, 1)
	outStreams.Store(obj.id, obj)
}

func freeOutStreamId(id int64) {
	outStreams.Delete(id)
}

//==========================
// InStream
//==========================

var inStreams sync.Map

func reserveInStreamId(obj *InStream) {
	obj.id = atomic.AddInt64(&seed, 1)
	inStreams.Store(obj.id, obj)
}

func freeInStreamId(id int64) {
	inStreams.Delete(id)
}

//==========================
// ExtractCallback
//==========================

var extractCallbacks sync.Map

func reserveExtractCallbackId(obj *ExtractCallback) {
	obj.id = atomic.AddInt64(&seed, 1)
	extractCallbacks.Store(obj.id, obj)
}

func freeExtractCallbackId(id int64) {
	extractCallbacks.Delete(id)
}
