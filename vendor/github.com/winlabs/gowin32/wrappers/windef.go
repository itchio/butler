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

package wrappers

const (
	MAX_PATH = 260
)

func MAKELONG(low uint16, high uint16) uint32 {
	return uint32(low) | (uint32(high) << 16)
}

func LOWORD(value uint32) uint16 {
	return uint16(value & 0x0000FFFF)
}

func HIWORD(value uint32) uint16 {
	return uint16((value >> 16) & 0x0000FFFF)
}
