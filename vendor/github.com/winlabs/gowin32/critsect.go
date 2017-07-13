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
)

type CriticalSection struct {
	nativeCriticalSection wrappers.CRITICAL_SECTION
}

func NewCriticalSection() *CriticalSection {
	cs := CriticalSection{}
	wrappers.InitializeCriticalSection(&cs.nativeCriticalSection)
	return &cs
}

func (self *CriticalSection) Close() error {
	wrappers.DeleteCriticalSection(&self.nativeCriticalSection)
	return nil
}

func (self *CriticalSection) Lock() {
	wrappers.EnterCriticalSection(&self.nativeCriticalSection)
}

func (self *CriticalSection) Unlock() {
	wrappers.LeaveCriticalSection(&self.nativeCriticalSection)
}

func (self *CriticalSection) TryLock() bool {
	return wrappers.TryEnterCriticalSection(&self.nativeCriticalSection)
}
