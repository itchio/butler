package sync_test

// func Example() {
// 	srcReader, _ := os.Open("content-v2.bin")
// 	defer srcReader.Close()
//
// 	rs := &sync.Context{}
//
// 	// here we store the whole signature in a byte slice,
// 	// but it could just as well be sent over a network connection for example
// 	sig := make([]sync.BlockHash, 0, 10)
// 	writeSignature := func(bl sync.BlockHash) error {
// 		sig = append(sig, bl)
// 		return nil
// 	}
//
// 	rs.CreateSignature(srcReader, writeSignature)
//
// 	targetReader, _ := os.Open("content-v1.bin")
//
// 	opsOut := make(chan sync.Operation)
// 	writeOperation := func(op sync.Operation) error {
// 		opsOut <- op
// 		return nil
// 	}
//
// 	go func() {
// 		defer close(opsOut)
// 		rs.InventPatch(targetReader, sig, writeOperation)
// 	}()
//
// 	srcWriter, _ := os.Open("content-v2-reconstructed.bin")
// 	srcReader.Seek(0, os.SEEK_SET)
//
// 	rs.ApplyPatch(srcWriter, srcReader, opsOut)
// }
