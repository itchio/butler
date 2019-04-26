package dmcunrar

/*
#include <stdint.h>
*/
import "C"

import (
	"sync"
	"sync/atomic"
)

var seed int64 = 1

//==============================
// FileReader
//==============================

var fileReaders sync.Map

func reserveFrId(obj *FileReader) {
	obj.id = atomic.AddInt64(&seed, 1)
	obj.opaque.id = C.int64_t(obj.id)
	fileReaders.Store(obj.id, obj)
}

func freeFrId(id int64) {
	fileReaders.Delete(id)
}

//==============================
// ExtractedFile
//==============================

var extractedFiles sync.Map

func reserveEfId(obj *ExtractedFile) {
	obj.id = atomic.AddInt64(&seed, 1)
	obj.opaque.id = C.int64_t(obj.id)
	extractedFiles.Store(obj.id, obj)
}

func freeEfId(id int64) {
	extractedFiles.Delete(id)
}
