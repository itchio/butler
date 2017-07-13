/*
 * Copyright (c) 2014-2016 MongoDB, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the license is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package gowin32

import (
	"github.com/winlabs/gowin32/wrappers"

	"unsafe"
)

type NetstatTCPState uint32

const (
	NetstatClosed      NetstatTCPState = wrappers.MIB_TCP_STATE_CLOSED
	NetstatListen      NetstatTCPState = wrappers.MIB_TCP_STATE_LISTEN
	NetstatSYNSent     NetstatTCPState = wrappers.MIB_TCP_STATE_SYN_SENT
	NetstatSYNReceived NetstatTCPState = wrappers.MIB_TCP_STATE_SYN_RCVD
	NetstatEstablished NetstatTCPState = wrappers.MIB_TCP_STATE_ESTAB
	NetstatFINWait1    NetstatTCPState = wrappers.MIB_TCP_STATE_FIN_WAIT1
	NetstatFINWait2    NetstatTCPState = wrappers.MIB_TCP_STATE_FIN_WAIT2
	NetstatCloseWait   NetstatTCPState = wrappers.MIB_TCP_STATE_CLOSE_WAIT
	NetstatClosing     NetstatTCPState = wrappers.MIB_TCP_STATE_CLOSING
	NetstatLastACK     NetstatTCPState = wrappers.MIB_TCP_STATE_LAST_ACK
	NetstatTimeWait    NetstatTCPState = wrappers.MIB_TCP_STATE_TIME_WAIT
	NetstatDeleteTCB   NetstatTCPState = wrappers.MIB_TCP_STATE_DELETE_TCB
)

type NetstatEntry struct {
	State         NetstatTCPState
	LocalAddress  string
	LocalPort     uint
	RemoteAddress string
	RemotePort    uint
}

func Netstat() ([]NetstatEntry, error) {
	var tcpTable wrappers.MIB_TCPTABLE
	bufPtr := (*byte)(unsafe.Pointer(&tcpTable))
	bufLength := uint32(unsafe.Sizeof(tcpTable))
	if err := wrappers.GetTcpTable(&tcpTable, &bufLength, true); err == wrappers.ERROR_INSUFFICIENT_BUFFER {
		buf := make([]byte, bufLength)
		bufPtr = &buf[0]
		if err := wrappers.GetTcpTable((*wrappers.MIB_TCPTABLE)(unsafe.Pointer(bufPtr)), &bufLength, true); err != nil {
			return nil, NewWindowsError("GetTcpTable", err)
		}
		wrappers.RtlMoveMemory((*byte)(unsafe.Pointer(&tcpTable)), bufPtr, unsafe.Sizeof(tcpTable))
	} else if err != nil {
		return nil, NewWindowsError("GetTcpTable", err)
	}
	bufPtr = (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(bufPtr)) + unsafe.Sizeof(tcpTable)))
	entries := []NetstatEntry{}
	for i := uint32(0); i < tcpTable.NumEntries; i++ {
		var tcpRow wrappers.MIB_TCPROW
		wrappers.RtlMoveMemory((*byte)(unsafe.Pointer(&tcpRow)), bufPtr, unsafe.Sizeof(tcpRow))
		entry := NetstatEntry{
			State:      NetstatTCPState(tcpRow.State),
			LocalPort:  uint(wrappers.Ntohs(uint16(tcpRow.LocalPort))),
			RemotePort: uint(wrappers.Ntohs(uint16(tcpRow.RemotePort))),
		}
		var err error
		if entry.LocalAddress, err = convertIPAddress(tcpRow.LocalAddr); err != nil {
			return nil, err
		}
		if entry.RemoteAddress, err = convertIPAddress(tcpRow.RemoteAddr); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		bufPtr = (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(bufPtr)) + unsafe.Sizeof(tcpRow)))
	}
	return entries, nil
}

func convertIPAddress(ipAddress uint32) (string, error) {
	buf := [16]uint16{}
	outbuf, err := wrappers.InetNtop(
		wrappers.AF_INET,
		(*byte)(unsafe.Pointer(&ipAddress)),
		&buf[0],
		16)
	if err != nil {
		return "", NewWindowsError("InetNtop", err)
	}
	return LpstrToString(outbuf), nil
}
