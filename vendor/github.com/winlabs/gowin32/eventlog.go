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

	"syscall"
)

type EventType uint32

const (
	EventTypeSuccess      EventType = wrappers.EVENTLOG_SUCCESS
	EventTypeError        EventType = wrappers.EVENTLOG_ERROR_TYPE
	EventTypeWarning      EventType = wrappers.EVENTLOG_WARNING_TYPE
	EventTypeInformation  EventType = wrappers.EVENTLOG_INFORMATION_TYPE
	EventTypeAuditSuccess EventType = wrappers.EVENTLOG_AUDIT_SUCCESS
	EventTypeAuditFailure EventType = wrappers.EVENTLOG_AUDIT_FAILURE
)

type EventSourceRegistration struct {
	SourceName           string
	CategoryCount        uint
	CategoryMessageFile  string
	EventMessageFile     string
	ParameterMessageFile string
	TypesSupported       EventType
}

func (self *EventSourceRegistration) Install() error {
	key, err := CreateRegKey(
		RegRootHKLM,
		"SYSTEM\\CurrentControlSet\\Services\\EventLog\\Application\\" + self.SourceName)
	if err != nil {
		return err
	}
	defer key.Close()
	if self.CategoryMessageFile != "" {
		if err := key.SetValueDWORD("CategoryCount", uint32(self.CategoryCount)); err != nil {
			return err
		}
		if err := key.SetValueString("CategoryMessageFile", self.CategoryMessageFile); err != nil {
			return err
		}
	}
	if self.EventMessageFile != "" {
		if err := key.SetValueString("EventMessageFile", self.EventMessageFile); err != nil {
			return err
		}
	}
	if self.ParameterMessageFile != "" {
		if err := key.SetValueString("ParameterMessageFile", self.ParameterMessageFile); err != nil {
			return err
		}
	}
	return key.SetValueDWORD("TypesSupported", uint32(self.TypesSupported))
}

type EventLogEvent struct {
	Type     EventType
	Category uint
	EventID  uint
	Strings  []string
	Data     []byte
}

type EventSource struct {
	handle syscall.Handle
}

func NewEventSource(sourceName string) (*EventSource, error) {
	hEventLog, err := wrappers.RegisterEventSource(nil, syscall.StringToUTF16Ptr(sourceName))
	if err != nil {
		return nil, NewWindowsError("RegisterEventSource", err)
	}
	return &EventSource{handle: hEventLog}, nil
}

func (self *EventSource) Close() error {
	if self.handle != 0 {
		if err := wrappers.DeregisterEventSource(self.handle); err != nil {
			return NewWindowsError("DeregisterEventSource", err)
		}
		self.handle = 0
	}
	return nil
}

func (self *EventSource) Report(event *EventLogEvent) error {
	var stringPtrsPtr **uint16
	var stringCount uint16
	if event.Strings != nil && len(event.Strings) > 0 {
		stringPtrsArray := make([]*uint16, len(event.Strings))
		for i, s := range event.Strings {
			stringPtrsArray[i] = syscall.StringToUTF16Ptr(s)
		}
		stringPtrsPtr = &stringPtrsArray[0]
		stringCount = uint16(len(event.Strings))
	}
	var data *byte
	var dataSize uint32
	if event.Data != nil && len(event.Data) > 0 {
		data = &event.Data[0]
		dataSize = uint32(len(event.Data))
	}
	err := wrappers.ReportEvent(
		self.handle,
		uint16(event.Type),
		uint16(event.Category),
		uint32(event.EventID),
		nil,
		stringCount,
		dataSize,
		stringPtrsPtr,
		data)
	if err != nil {
		return NewWindowsError("ReportEvent", err)
	}
	return nil
}

func (self *EventSource) Error(eventID uint, strings ...string) error {
	return self.Report(&EventLogEvent{
		Type:    EventTypeError,
		EventID: eventID,
		Strings: strings,
	})
}

func (self *EventSource) Warning(eventID uint, strings ...string) error {
	return self.Report(&EventLogEvent{
		Type:    EventTypeWarning,
		EventID: eventID,
		Strings: strings,
	})
}

func (self *EventSource) Info(eventID uint, strings ...string) error {
	return self.Report(&EventLogEvent{
		Type:    EventTypeInformation,
		EventID: eventID,
		Strings: strings,
	})
}
