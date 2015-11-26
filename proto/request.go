package proto

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
)

func (c *Conn) SendRequest(name string, wantReply bool, payload interface{}) (bool, interface{}, error) {
	var payloadBytes []byte = nil
	if payload != nil {
		payloadBuf := new(bytes.Buffer)
		err := gob.NewEncoder(payloadBuf).Encode(&payload)
		if err != nil {
			return false, nil, err
		}
		payloadBytes = payloadBuf.Bytes()
	}

	status, replyBytes, err := c.Conn.SendRequest(name, wantReply, payloadBytes)
	if err != nil {
		err = fmt.Errorf("in sendrequest(%s): %s", name, err.Error())
		return false, nil, err
	}

	var reply interface{} = nil
	if len(replyBytes) > 0 {
		err := gob.NewDecoder(bytes.NewReader(replyBytes)).Decode(&reply)
		if err != nil {
			log.Println("when parsing reply")
			return false, nil, err
		}
	}

	return status, reply, nil
}

func (c *Conn) Blog(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	_, _, err := c.SendRequest("butler/log", false, LogEntry{Message: msg})
	return err
}
