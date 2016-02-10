package main

import "regexp"

func push(src string, repoSpec string) {
	must(doPush(src, repoSpec))
}

var (
	addressRe   = regexp.MustCompile(`[^:]:.*`)
	defaultPort = 22
)

func doPush(src string, repoSpec string) error {
	// 	address := *pushArgs.address
	// 	if !addressRe.MatchString(address) {
	// 		address = fmt.Sprintf("%s:%d", address, defaultPort)
	// 	}
	//
	// 	Logf("Connecting to %s", address)
	// 	conn, err := proto.Connect(address, *pushArgs.identity, version)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	defer conn.Close()
	// 	Logf("Connected to %s", conn.Conn.RemoteAddr())
	//
	// 	closed := make(chan bool)
	// 	var wg sync.WaitGroup
	//
	// 	wg.Add(1)
	// 	go func() {
	// 		for req := range conn.Reqs {
	// 			payload, err := proto.Unmarshal(req.Payload)
	// 			if err != nil {
	// 				log.Printf("Error receiving payload: %s\n", err.Error())
	// 				conn.Close()
	// 			}
	//
	// 			switch v := payload.(type) {
	// 			case proto.LogEntry:
	// 				log.Printf("remote: %s\n", v.Message)
	// 			case proto.EndOfSources:
	// 				wg.Done()
	// 			default:
	// 				log.Printf("Server sent us unknown req %s\n", req.Type)
	// 			}
	// 		}
	// 	}()
	//
	// 	go func() {
	// 		for req := range conn.Chans {
	// 			req := req
	//
	// 			payload, err := proto.Unmarshal(req.ExtraData())
	// 			if err != nil {
	// 				log.Printf("error while decoding chanreq %s payload: %s\n", req.ChannelType(), err)
	// 				conn.Close()
	// 				return
	// 			}
	//
	// 			switch v := payload.(type) {
	// 			case proto.SourceFile:
	// 				wg.Add(1)
	// 				go func() {
	// 					defer wg.Done()
	// 					err := handleSourceFile(src, conn, req, v)
	// 					if err != nil {
	// 						panic(err)
	// 					}
	// 				}()
	// 			default:
	// 				log.Printf("unknown channel type req'd from remote: %s\n", req.ChannelType())
	// 				req.Reject(ssh.UnknownChannelType, "")
	// 			}
	//
	// 			if err != nil {
	// 				log.Printf("Error while handling channel :%s\n", err.Error())
	// 				conn.Close()
	// 				return
	// 			}
	// 		}
	// 		log.Println("done handling chans, closing")
	// 		close(closed)
	// 	}()
	//
	// 	up := proto.Target{RepoSpec: repoSpec}
	// 	ok, _, err := conn.SendRequest("wharf/target", true, up)
	// 	if err != nil {
	// 		return fmt.Errorf("Could not send target: %s", err.Error())
	// 	}
	//
	// 	if !ok {
	// 		return fmt.Errorf("Could not find upload to replace from '%s'", repoSpec)
	// 	}
	// 	Log("Locked onto upload target")
	//
	// 	done := make(chan bool)
	// 	errs := make(chan error)
	//
	// 	var totalSize uint64
	//
	// 	fileList := make(map[string]os.FileInfo)
	//
	// 	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
	// 		if err != nil {
	// 			errs <- err
	// 			return err
	// 		}
	//
	// 		if info.IsDir() && len(path) > 1 && strings.HasPrefix(info.Name(), ".") {
	// 			log.Printf("Skipping %s because it starts with .\n", path)
	// 			return filepath.SkipDir
	// 		}
	//
	// 		if !info.IsDir() {
	// 			fileList[path] = info
	// 		}
	// 		return nil
	// 	})
	//
	// 	for _, info := range fileList {
	// 		fileSize := uint64(info.Size())
	// 		totalSize += fileSize
	// 	}
	// 	log.Printf("Have %s in %d files\n", humanize.Bytes(totalSize), len(fileList))
	//
	// 	go func() {
	// 		wg.Wait()
	// 		done <- true
	// 	}()
	//
	// 	select {
	// 	case err := <-errs:
	// 		return err
	// 	case <-done:
	// 		return nil
	// 	case <-closed:
	// 		return errors.New("remote closed the connection")
	// 	}
	// }
	//
	// func handleSourceFile(src string, conn *proto.Conn, req ssh.NewChannel, sf proto.SourceFile) (err error) {
	// 	ch, reqs, err := req.Accept()
	// 	if err != nil {
	// 		return
	// 	}
	// 	defer ch.Close()
	// 	go ssh.DiscardRequests(reqs)
	//
	// 	br := dec.NewBrotliReader(ch)
	// 	defer br.Close()
	// 	sig := make([]rsync.BlockHash, 0)
	//
	// 	gdec := gob.NewDecoder(br)
	// 	var recipient interface{}
	//
	// 	for {
	// 		err := gdec.Decode(&recipient)
	// 		if err != nil {
	// 			if err == io.EOF {
	// 				break
	// 			}
	// 			return err
	// 		}
	//
	// 		switch v := recipient.(type) {
	// 		case rsync.BlockHash:
	// 			sig = append(sig, v)
	// 		case proto.MD5Hash:
	// 			_ = v.Hash
	// 		default:
	// 			return fmt.Errorf("wat")
	// 		}
	// 	}
	//
	// 	// create delta + send it
	//
	// 	path := path.Join(src, sf.Path)
	//
	// 	fr, err := os.Open(path)
	// 	if err != nil {
	// 		log.Printf("warn: could not open %s - deleted?\n", path)
	// 		return nil
	// 	}
	// 	defer fr.Close()
	//
	// 	out, err := conn.OpenChannel("wharf/patch-file", &proto.FilePatched{
	// 		Path: path,
	// 	})
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	h := md5.New()
	// 	var newMD5 []byte
	//
	// 	err = func() (err error) {
	// 		defer out.CloseWrite()
	//
	// 		opWriter := func(op rsync.Operation) error {
	// 			return out.Send(op)
	// 		}
	//
	// 		tr := io.TeeReader(fr, h)
	//
	// 		rs := &rsync.RSync{}
	// 		err = rs.InventRecipe(tr, sig, opWriter)
	// 		if err != nil {
	// 			return fmt.Errorf("while creating delta for %s: %s", path, err.Error())
	// 		}
	//
	// 		newMD5 = h.Sum(nil)
	// 		out.Send(proto.MD5Hash{Hash: newMD5})
	// 		return
	// 	}()
	//
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	// wait for read end to close
	// 	out.AwaitEOF()
	//
	// 	log.Printf("%8s | %x | %s", humanize.Bytes(uint64(out.BytesWritten())), newMD5, path)
	//
	// 	return
	return nil
}
