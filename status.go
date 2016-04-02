package main

import "fmt"

func status(target string) {
	must(doStatus(target))
}

func doStatus(target string) error {
	client, err := authenticateViaWharf()
	if err != nil {
		return err
	}

	listChannelsResp, err := client.ListChannels(target)
	if err != nil {
		return err
	}

	for _, ch := range listChannelsResp.Channels {
		if ch.Head != nil {
			files := ch.Head.Files
			fmt.Printf("Chan %s: patch %s | signature %s | archive %s\n", ch.Name,
				files.Patch.State, files.Signature.State, files.Archive.State)
		} else {
			fmt.Printf("Chan %s: no head\n")
		}
	}

	return nil
}
