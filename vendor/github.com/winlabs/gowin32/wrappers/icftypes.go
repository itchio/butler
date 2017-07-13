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
	NET_FW_PROFILE_DOMAIN   = 0
	NET_FW_PROFILE_STANDARD = 1
	NET_FW_PROFILE_CURRENT  = 2
	NET_FW_PROFILE_TYPE_MAX = 3
)

const (
	NET_FW_IP_VERSION_V4  = 0
	NET_FW_IP_VERSION_V6  = 1
	NET_FW_IP_VERSION_ANY = 2
	NET_FW_IP_VERSION_MAX = 3
)

const (
	NET_FW_PROTOCOL_TCP = 6
	NET_FW_PROTOCOL_UDP = 17
	NET_FW_PROTOCOL_ANY = 256
)

const (
	NET_FW_RULE_DIR_IN  = 1
	NET_FW_RULE_DIR_OUT = 2
	NET_FW_RULE_DIR_MAX = 3
)

const (
	NET_FW_ACTION_BLOCK = 0
	NET_FW_ACTION_ALLOW = 1
	NET_FW_ACTION_MAX   = 2
)
