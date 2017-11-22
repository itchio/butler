package sz

type ReadStats struct {
	Reads []ReadOp
}

type ReadOp struct {
	Offset int64
	Size   int64
}

func (rs *ReadStats) RecordRead(offset int64, size int64) {
	rs.Reads = append(rs.Reads, ReadOp{
		Offset: offset,
		Size:   size,
	})
}
