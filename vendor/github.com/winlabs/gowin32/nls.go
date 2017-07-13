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

type LocaleNameFlags uint32

const (
	LocaleNameAllowNeutralNames LocaleNameFlags = wrappers.LOCALE_ALLOW_NEUTRAL_NAMES
)

type Language uint16

var (
	LanguageSystemDefault = Language(wrappers.LANG_SYSTEM_DEFAULT)
	LanguageUserDefault   = Language(wrappers.LANG_USER_DEFAULT)
)

type Locale   uint32

var (
	LocaleSystemDefault     = Locale(wrappers.LOCALE_SYSTEM_DEFAULT)
	LocaleUserDefault       = Locale(wrappers.LOCALE_USER_DEFAULT)
	LocaleCustomDefault     = Locale(wrappers.LOCALE_CUSTOM_DEFAULT)
	LocaleCustomUnspecified = Locale(wrappers.LOCALE_CUSTOM_UNSPECIFIED)
	LocaleCustomUIDefault   = Locale(wrappers.LOCALE_CUSTOM_UI_DEFAULT)
	LocaleNeutral           = Locale(wrappers.LOCALE_NEUTRAL)
	LocaleInvariant         = Locale(wrappers.LOCALE_INVARIANT)
)

func (locale Locale) Language() Language {
	return Language(wrappers.LANGIDFROMLCID(uint32(locale)))
}

func LocaleFromLocaleName(localeName string, flags LocaleNameFlags) (Locale, error) {
	lcid, err := wrappers.LocaleNameToLCID(
		syscall.StringToUTF16Ptr(localeName),
		uint32(flags))
	if err != nil {
		return 0, NewWindowsError("LocaleNameToLCID", err)
	}
	return Locale(lcid), nil
}
