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

import (
	"syscall"
	"unsafe"
)

const (
	ANYSIZE_ARRAY = 1
)

type LUID struct {
	LowPart  uint32
	HighPart int32
}

type LIST_ENTRY struct {
	Flink *LIST_ENTRY
	Blink *LIST_ENTRY
}

const (
	VER_SUITE_SMALLBUSINESS            = 0x00000001
	VER_SUITE_ENTERPRISE               = 0x00000002
	VER_SUITE_BACKOFFICE               = 0x00000004
	VER_SUITE_COMMUNICATIONS           = 0x00000008
	VER_SUITE_TERMINAL                 = 0x00000010
	VER_SUITE_SMALLBUSINESS_RESTRICTED = 0x00000020
	VER_SUITE_EMBEDDEDNT               = 0x00000040
	VER_SUITE_DATACENTER               = 0x00000080
	VER_SUITE_SINGLEUSERTS             = 0x00000100
	VER_SUITE_PERSONAL                 = 0x00000200
	VER_SUITE_BLADE                    = 0x00000400
	VER_SUITE_EMBEDDED_RESTRICTED      = 0x00000800
	VER_SUITE_SECURITY_APPLIANCE       = 0x00001000
	VER_SUITE_STORAGE_SERVER           = 0x00002000
	VER_SUITE_COMPUTE_SERVER           = 0x00004000
	VER_SUITE_WH_SERVER                = 0x00008000
)

const (
	LANG_NEUTRAL   = 0x00
	LANG_INVARIANT = 0x7F

	LANG_AFRIKAANS     = 0x36
	LANG_ALBANIAN      = 0x1C
	LANG_ALSATIAN      = 0x84
	LANG_AMHARIC       = 0x5E
	LANG_ARABIC        = 0x01
	LANG_ARMENIAN      = 0x2B
	LANG_ASSAMESE      = 0x4D
	LANG_AZERI         = 0x2C
	LANG_BASHKIR       = 0x6D
	LANG_BASQUE        = 0x2D
	LANG_BELARUSIAN    = 0x23
	LANG_BENGALI       = 0x45
	LANG_BRETON        = 0x7E
	LANG_BOSNIAN       = 0x1A
	LANG_BULGARIAN     = 0x02
	LANG_CATALAN       = 0x03
	LANG_CHINESE       = 0x04
	LANG_CORSICAN      = 0x83
	LANG_CROATIAN      = 0x1A
	LANG_CZECH         = 0x05
	LANG_DANISH        = 0x06
	LANG_DARI          = 0x8C
	LANG_DIVEHI        = 0x65
	LANG_DUTCH         = 0x13
	LANG_ENGLISH       = 0x09
	LANG_ESTONIAN      = 0x25
	LANG_FAEROESE      = 0x38
	LANG_FARSI         = 0x29
	LANG_FILIPINO      = 0x64
	LANG_FINNISH       = 0x0B
	LANG_FRENCH        = 0x0C
	LANG_FRISIAN       = 0x62
	LANG_GALICIAN      = 0x56
	LANG_GEORGIAN      = 0x37
	LANG_GERMAN        = 0x07
	LANG_GREEK         = 0x08
	LANG_GREENLANDIC   = 0x6F
	LANG_GUJARATI      = 0x47
	LANG_HAUSA         = 0x68
	LANG_HEBREW        = 0x0D
	LANG_HINDI         = 0x39
	LANG_HUNGARIAN     = 0x0E
	LANG_ICELANDIC     = 0x0F
	LANG_IGBO          = 0x70
	LANG_INDONESIAN    = 0x21
	LANG_INUKTITUT     = 0x5D
	LANG_IRISH         = 0x3C
	LANG_ITALIAN       = 0x10
	LANG_JAPANESE      = 0x11
	LANG_KANNADA       = 0x4B
	LANG_KASHMIRI      = 0x60
	LANG_KAZAK         = 0x3F
	LANG_KHMER         = 0x53
	LANG_KICHE         = 0x86
	LANG_KINYARWANDA   = 0x87
	LANG_KONKANI       = 0x57
	LANG_KOREAN        = 0x12
	LANG_KYRGYZ        = 0x40
	LANG_LAO           = 0x54
	LANG_LATVIAN       = 0x26
	LANG_LITHUANIAN    = 0x27
	LANG_LOWER_SORBIAN = 0x2E
	LANG_LUXEMBOURGISH = 0x6E
	LANG_MACEDONIAN    = 0x2F
	LANG_MALAY         = 0x3E
	LANG_MALAYALAM     = 0x4C
	LANG_MALTESE       = 0x3A
	LANG_MANIPURI      = 0x58
	LANG_MAORI         = 0x81
	LANG_MAPUDUNGUN    = 0x7A
	LANG_MARATHI       = 0x4E
	LANG_MOHAWK        = 0x7C
	LANG_MONGOLIAN     = 0x50
	LANG_NEPALI        = 0x61
	LANG_NORWEGIAN     = 0x14
	LANG_OCCITAN       = 0x82
	LANG_ORIYA         = 0x48
	LANG_PASHTO        = 0x63
	LANG_PERSIAN       = 0x29
	LANG_POLISH        = 0x15
	LANG_PORTUGUESE    = 0x16
	LANG_PUNJABI       = 0x46
	LANG_QUECHUA       = 0x6B
	LANG_ROMANIAN      = 0x18
	LANG_ROMANSH       = 0x17
	LANG_RUSSIAN       = 0x19
	LANG_SAMI          = 0x3B
	LANG_SANSKRIT      = 0x4F
	LANG_SERBIAN       = 0x1F
	LANG_SINDHI        = 0x59
	LANG_SINHALESE     = 0x5B
	LANG_SLOVAK        = 0x1B
	LANG_SLOVENIAN     = 0x24
	LANG_SOTHO         = 0x6C
	LANG_SPANISH       = 0x0A
	LANG_SWAHILI       = 0x41
	LANG_SWEDISH       = 0x1D
	LANG_SYRIAC        = 0x5A
	LANG_TAJIK         = 0x28
	LANG_TAMAZIGHT     = 0x5F
	LANG_TAMIL         = 0x49
	LANG_TATAR         = 0x44
	LANG_TELUGU        = 0x4A
	LANG_THAI          = 0x1E
	LANG_TIBETAN       = 0x51
	LANG_TIGRIGNA      = 0x73
	LANG_TSWANA        = 0x32
	LANG_TURKISH       = 0x1F
	LANG_TURKMEN       = 0x42
	LANG_UIGHUR        = 0x80
	LANG_UKRAINIAN     = 0x22
	LANG_UPPER_SORBIAN = 0x2E
	LANG_URDU          = 0x20
	LANG_UZBEK         = 0x43
	LANG_VIETNAMESE    = 0x2A
	LANG_WELSH         = 0x52
	LANG_WOLOF         = 0x88
	LANG_XHOSA         = 0x34
	LANG_YAKUT         = 0x85
	LANG_YI            = 0x78
	LANG_YORUBA        = 0x6A
	LANG_ZULU          = 0x35
)

const (
	SUBLANG_NEUTRAL            = 0x00
	SUBLANG_DEFAULT            = 0x01
	SUBLANG_SYS_DEFAULT        = 0x02
	SUBLANG_CUSTOM_DEFAULT     = 0x03
	SUBLANG_CUSTOM_UNSPECIFIED = 0x04
	SUBLANG_UI_CUSTOM_DEFAULT  = 0x05

	SUBLANG_AFRIKAANS_SOUTH_AFRICA              = 0x01
	SUBLANG_ALBANIAN_ALBANIA                    = 0x01
	SUBLANG_ALSATIAN_FRANCE                     = 0x01
	SUBLANG_AMHARIC_ETHIOPIA                    = 0x01
	SUBLANG_ARABIC_SAUDI_ARABIA                 = 0x01
	SUBLANG_ARABIC_IRAQ                         = 0x02
	SUBLANG_ARABIC_EGYPT                        = 0x03
	SUBLANG_ARABIC_LIBYA                        = 0x04
	SUBLANG_ARABIC_ALGERIA                      = 0x05
	SUBLANG_ARABIC_MOROCCO                      = 0x06
	SUBLANG_ARABIC_TUNISIA                      = 0x07
	SUBLANG_ARABIC_OMAN                         = 0x08
	SUBLANG_ARABIC_YEMEN                        = 0x09
	SUBLANG_ARABIC_SYRIA                        = 0x0A
	SUBLANG_ARABIC_JORDAN                       = 0x0B
	SUBLANG_ARABIC_LEBANON                      = 0x0C
	SUBLANG_ARABIC_KUWAIT                       = 0x0D
	SUBLANG_ARABIC_UAE                          = 0x0E
	SUBLANG_ARABIC_BAHRAIN                      = 0x0F
	SUBLANG_ARABIC_QATAR                        = 0x10
	SUBLANG_ARMENIAN_ARMENIA                    = 0x01
	SUBLANG_ASSAMESE_INDIA                      = 0x01
	SUBLANG_AZERI_LATIN                         = 0x01
	SUBLANG_AZERI_CYRILLIC                      = 0x01
	SUBLANG_BASHKIR_RUSSIA                      = 0x01
	SUBLANG_BASQUE_BASQUE                       = 0x01
	SUBLANG_BELARUSIAN_BELARUS                  = 0x01
	SUBLANG_BENGALI_INDIA                       = 0x01
	SUBLANG_BENGALI_BANGLADESH                  = 0x02
	SUBLANG_BOSNIAN_BOSNIA_HERZEGOVINA_LATIN    = 0x05
	SUBLANG_BOSNIAN_BOSNIA_HERZEGOVINA_CYRILLIC = 0x08
	SUBLANG_BRETON_FRANCE                       = 0x01
	SUBLANG_BULGARIAN_BULGARIA                  = 0x01
	SUBLANG_CATALAN_CATALAN                     = 0x01
	SUBLANG_CHINESE_TRADITIONAL                 = 0x01
	SUBLANG_CHINESE_SIMPLIFIED                  = 0x02
	SUBLANG_CHINESE_HONGKONG                    = 0x03
	SUBLANG_CHINESE_SINGAPORE                   = 0x04
	SUBLANG_CHINESE_MACAU                       = 0x05
	SUBLANG_CORSICAN_FRANCE                     = 0x01
	SUBLANG_CZECH_CZECH_REPUBLIC                = 0x01
	SUBLANG_CROATIAN_CROATIA                    = 0x01
	SUBLANG_CROATIAN_BOSNIA_HERVEGOVINA_LATIN   = 0x04
	SUBLANG_DANISH_DENMARK                      = 0x01
	SUBLANG_DARI_AFGHANISTAN                    = 0x01
	SUBLANG_DIVEHI_MALDIVES                     = 0x01
	SUBLANG_DUTCH                               = 0x01
	SUBLANG_DUTCH_BELGIAN                       = 0x02
	SUBLANG_ENGLISH_US                          = 0x01
	SUBLANG_ENGLISH_UK                          = 0x02
	SUBLANG_ENGLISH_AUS                         = 0x03
	SUBLANG_ENGLISH_CAN                         = 0x04
	SUBLANG_ENGLISH_NZ                          = 0x05
	SUBLANG_ENGLISH_EIRE                        = 0x06
	SUBLANG_ENGLISH_SOUTH_AFRICA                = 0x07
	SUBLANG_ENGLISH_JAMAICA                     = 0x08
	SUBLANG_ENGLISH_CARIBBEAN                   = 0x09
	SUBLANG_ENGLISH_BELIZE                      = 0x0A
	SUBLANG_ENGLISH_TRINIDAD                    = 0x0B
	SUBLANG_ENGLISH_ZIMBABWE                    = 0x0C
	SUBLANG_ENGLISH_PHILIPPINES                 = 0x0D
	SUBLANG_ENGLISH_INDIA                       = 0x10
	SUBLANG_ENGLISH_MALAYSIA                    = 0x11
	SUBLANG_ENGLISH_SINGAPORE                   = 0x12
	SUBLANG_ESTONIAN_ESTONIA                    = 0x01
	SUBLANG_FAEROESE_FAERO_ISLANDS              = 0x01
	SUBLANG_FILIPINO_PHILIPPINES                = 0x01
	SUBLANG_FINNISH_FINLAND                     = 0x01
	SUBLANG_FRENCH                              = 0x01
	SUBLANG_FRENCH_BELGIAN                      = 0x02
	SUBLANG_FRENCH_CANADIAN                     = 0x03
	SUBLANG_FRENCH_SWISS                        = 0x04
	SUBLANG_FRENCH_LUXEMBOURG                   = 0x05
	SUBLANG_FRENCH_MONACO                       = 0x06
	SUBLANG_FRISIAN_NETHERLANDS                 = 0x01
	SUBLANG_GALICIAN_GALICIAN                   = 0x01
	SUBLANG_GEORGIAN_GEORGIA                    = 0x01
	SUBLANG_GERMAN                              = 0x01
	SUBLANG_GERMAN_SWISS                        = 0x02
	SUBLANG_GERMAN_AUSTRIAN                     = 0x03
	SUBLANG_GERMAN_LUXEMBOURG                   = 0x04
	SUBLANG_GERMAN_LIECHTENSTEIN                = 0x05
	SUBLANG_GREEK_GREECE                        = 0x01
	SUBLANG_GREENLANDIC_GREENLAND               = 0x02
	SUBLANG_GUJARATI_INDIA                      = 0x01
	SUBLANG_HAUSA_NIGERIA_LATIN                 = 0x01
	SUBLANG_HEBREW_ISRAEL                       = 0x01
	SUBLANG_HINDI_INDIA                         = 0x01
	SUBLANG_HUNGARIAN_HUNGARY                   = 0x01
	SUBLANG_ICELANDIC_ICELAND                   = 0x01
	SUBLANG_IGBO_NIGERIA                        = 0x01
	SUBLANG_INDONESIAN_INDONESIA                = 0x01
	SUBLANG_INUKTITUT_CANADA                    = 0x01
	SUBLANG_INUKTITUT_CANADA_LATIN              = 0x02
	SUBLANG_IRISH_IRELAND                       = 0x02
	SUBLANG_ITALIAN                             = 0x01
	SUBLANG_ITALIAN_SWISS                       = 0x02
	SUBLANG_JAPANESE_JAPAN                      = 0x01
	SUBLANG_KANNADA_INDIA                       = 0x01
	SUBLANG_KASHMIRI_SASIA                      = 0x02
	SUBLANG_KASHMIRI_INDIA                      = 0x02
	SUBLANG_KAZAK_KAZAKHSTAN                    = 0x01
	SUBLANG_KHMER_CAMBODIA                      = 0x01
	SUBLANG_KICHE_GUATEMALA                     = 0x01
	SUBLANG_KINYARWANDA_RWANDA                  = 0x01
	SUBLANG_KONKANI_INDIA                       = 0x01
	SUBLANG_KOREAN                              = 0x01
	SUBLANG_KYRGYZ_KYRGYZSTAN                   = 0x01
	SUBLANG_LAO_LAO                             = 0x01
	SUBLANG_LATVIAN_LATVIA                      = 0x01
	SUBLANG_LITHUANIAN                          = 0x01
	SUBLANG_LOWER_SORBIAN_GERMANY               = 0x02
	SUBLANG_LUXEMBOURGISH_LUXEMBOURG            = 0x01
	SUBLANG_MACEDONIAN_MACEDONIA                = 0x01
	SUBLANG_MALAY_MALAYSIA                      = 0x01
	SUBLANG_MALAY_BRUNEI_DARUSSALAM             = 0x02
	SUBLANG_MALAYALAM_INDIA                     = 0x01
	SUBLANG_MALTESE_MALTA                       = 0x01
	SUBLANG_MAORI_NEW_ZEALAND                   = 0x01
	SUBLANG_MAPUDUNGUN_CHILE                    = 0x01
	SUBLANG_MARATHI_INDIA                       = 0x01
	SUBLANG_MOHAWK_MOHAWK                       = 0x01
	SUBLANG_MONGOLIAN_CYRILLIC_MONGOLIA         = 0x01
	SUBLANG_MONGOLIAN_PRC                       = 0x02
	SUBLANG_NEPALI_INDIA                        = 0x02
	SUBLANG_NEPALI_NEPAL                        = 0x01
	SUBLANG_NORWEGIAN_BOKMAL                    = 0x01
	SUBLANG_NORWEGIAN_NYNORSK                   = 0x02
	SUBLANG_OCCITAN_FRANCE                      = 0x01
	SUBLANG_ORIYA_INDIA                         = 0x01
	SUBLANG_PASHTO_AFGHANISTAN                  = 0x01
	SUBLANG_PERSIAN_IRAN                        = 0x01
	SUBLANG_POLISH_POLAND                       = 0x01
	SUBLANG_PORTUGUESE                          = 0x02
	SUBLANG_PORTUGUESE_BRAZILIAN                = 0x01
	SUBLANG_PUNJABI_INDIA                       = 0x01
	SUBLANG_QUECHUA_BOLIVIA                     = 0x01
	SUBLANG_QUECHUA_ECUADOR                     = 0x02
	SUBLANG_QUECHUA_PERU                        = 0x03
	SUBLANG_ROMANIAN_ROMANIA                    = 0x01
	SUBLANG_ROMANSH_SWITZERLAND                 = 0x01
	SUBLANG_RUSSIAN_RUSSIA                      = 0x01
	SUBLANG_SAMI_NORTHERN_NORWAY                = 0x01
	SUBLANG_SAMI_NORTHERN_SWEDEN                = 0x02
	SUBLANG_SAMI_NORTHERN_FINLAND               = 0x03
	SUBLANG_SAMI_LULE_NORWAY                    = 0x04
	SUBLANG_SAMI_LULE_SWEDEN                    = 0x05
	SUBLANG_SAMI_SOUTHERN_NORWAY                = 0x06
	SUBLANG_SAMI_SOUTHERN_SWEDEN                = 0x07
	SUBLANG_SAMI_SKOLT_FINLAND                  = 0x08
	SUBLANG_SAMI_INARI_FINLAND                  = 0x09
	SUBLANG_SANSKRIT_INDIA                      = 0x01
	SUBLANG_SERBIAN_BOSNIA_HERZEGOVINA_LATIN    = 0x06
	SUBLANG_SERBIAN_BOSNIA_HERZEGOVINA_CYRILLIC = 0x07
	SUBLANG_SERBIAN_CROATIA                     = 0x01
	SUBLANG_SERBIAN_LATIN                       = 0x02
	SUBLANG_SERBIAN_CYRILLIC                    = 0x03
	SUBLANG_SINDHI_INDIA                        = 0x01
	SUBLANG_SINDHI_PAKISTAN                     = 0x02
	SUBLANG_SINDHI_AFGHANISTAN                  = 0x02
	SUBLANG_SINHALESE_SRI_LANKA                 = 0x01
	SUBLANG_SOTHO_NORTHERN_SOUTH_AFRICA         = 0x01
	SUBLANG_SLOVAK_SLOVAKIA                     = 0x01
	SUBLANG_SLOVENIAN_SLOVENIA                  = 0x01
	SUBLANG_SPANISH                             = 0x01
	SUBLANG_SPANISH_MEXICAN                     = 0x02
	SUBLANG_SPANISH_MODERN                      = 0x03
	SUBLANG_SPANISH_GUATEMALA                   = 0x04
	SUBLANG_SPANISH_COSTA_RICA                  = 0x05
	SUBLANG_SPANISH_PANAMA                      = 0x06
	SUBLANG_SPANISH_DOMINICAN_REPUBLIC          = 0x07
	SUBLANG_SPANISH_VENEZUELA                   = 0x08
	SUBLANG_SPANISH_COLOMBIA                    = 0x09
	SUBLANG_SPANISH_PERU                        = 0x0A
	SUBLANG_SPANISH_ARGENTINA                   = 0x0B
	SUBLANG_SPANISH_ECUADOR                     = 0x0C
	SUBLANG_SPANISH_CHILE                       = 0x0D
	SUBLANG_SPANISH_URUGUAY                     = 0x0E
	SUBLANG_SPANISH_PARAGUAY                    = 0x0F
	SUBLANG_SPANISH_BOLIVIA                     = 0x10
	SUBLANG_SPANISH_EL_SALVADOR                 = 0x11
	SUBLANG_SPANISH_HONDURAS                    = 0x12
	SUBLANG_SPANISH_NICARAGUA                   = 0x13
	SUBLANG_SPANISH_PEURTO_RICO                 = 0x14
	SUBLANG_SPANISH_US                          = 0x15
	SUBLANG_SWEDISH                             = 0x01
	SUBLANG_SWEDISH_FINLAND                     = 0x02
	SUBLANG_SYRIAC_SYRIA                        = 0x01
	SUBLANG_TAJIK_TAJIKISTAN                    = 0x01
	SUBLANG_TAMAZIGHT_ALGERIA_LATIN             = 0x02
	SUBLANG_TAMIL_INDIA                         = 0x01
	SUBLANG_TATAR_RUSSIA                        = 0x01
	SUBLANG_TELUGU_INDIA                        = 0x01
	SUBLANG_THAI_THAILAND                       = 0x01
	SUBLANG_TIBETAN_PRC                         = 0x01
	SUBLANG_TIGRIGNA_ERITREA                    = 0x02
	SUBLANG_TSWANA_SOUTH_AFRICA                 = 0x01
	SUBLANG_TURKISH_TURKEY                      = 0x01
	SUBLANG_TURKMEN_TURKMENISTAN                = 0x01
	SUBLANG_UIGHUR_PRC                          = 0x01
	SUBLANG_UKRAINIAN_UKRAINE                   = 0x01
	SUBLANG_UPPER_SORBIAN_GERMANY               = 0x01
	SUBLANG_URDU_PAKISTAN                       = 0x01
	SUBLANG_URDU_INDIA                          = 0x02
	SUBLANG_UZBEK_LATIN                         = 0x01
	SUBLANG_UZBEK_CYRILLIC                      = 0x02
	SUBLANG_VIETNAMESE_VIETNAM                  = 0x01
	SUBLANG_WELSH_UNITED_KINGDOM                = 0x01
	SUBLANG_WOLOF_SENEGAL                       = 0x01
	SUBLANG_XHOSA_SOUTH_AFRICA                  = 0x01
	SUBLANG_YAKUT_RUSSIA                        = 0x01
	SUBLANG_YI_PRC                              = 0x01
	SUBLANG_YORUBA_NIGERIA                      = 0x01
	SUBLANG_ZULU_SOUTH_AFRICA                   = 0x01
)

const (
	SORT_DEFAULT                = 0x0
	SORT_JAPANESE_XJIS          = 0x0
	SORT_JAPANESE_UNICODE       = 0x1
	SORT_JAPANESE_RADICALSTROKE = 0x4
	SORT_CHINESE_BIG5           = 0x0
	SORT_CHINESE_PRCP           = 0x0
	SORT_CHINESE_UNICODE        = 0x1
	SORT_CHINESE_PRC            = 0x2
	SORT_CHINESE_BOPOMOFO       = 0x3
	SORT_CHINESE_RADICALSTROKE  = 0x4
	SORT_KOREAN_KSC             = 0x0
	SORT_KOREAN_UNICODE         = 0x1
	SORT_GERMAN_PHONE_BOOK      = 0x1
	SORT_HUNGARIAN_DEFAULT      = 0x0
	SORT_HUNGARIAN_TECHNICAL    = 0x1
	SORT_GEORGIAN_TRADITIONAL   = 0x0
	SORT_GEORGIAN_MODERN        = 0x1
)

func MAKELCID(languageID uint16, sortID uint16) uint32 {
	return (uint32)(sortID) << 16 | uint32(languageID)
}

func MAKESORTLCID(languageID uint16, sortID uint16, sortVersion uint16) uint32 {
	return MAKELCID(languageID, sortID) | (uint32)(sortVersion) << 20
}

func LANGIDFROMLCID(lcid uint32) uint16 {
	return uint16(lcid)
}

func SORTIDFROMLCID(lcid uint32) uint16 {
	return uint16((lcid >> 16) & 0xF)
}

func SORTVERSIONFROMLCID(lcid uint32) uint16 {
	return uint16((lcid >> 20) & 0xF)
}

func MAKELANGID(primaryLanguage uint16, subLanguage uint16) uint16 {
	return (subLanguage << 10) | primaryLanguage
}

func PRIMARYLANGID(lgid uint16) uint16 {
	return lgid & 0x03FF
}

func SUBLANGID(lgid uint16) uint16 {
	return lgid >> 10
}

var (
	LANG_SYSTEM_DEFAULT = MAKELANGID(LANG_NEUTRAL, SUBLANG_SYS_DEFAULT)
	LANG_USER_DEFAULT   = MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT)
)

var (
	LOCALE_SYSTEM_DEFAULT     = MAKELCID(LANG_SYSTEM_DEFAULT, SORT_DEFAULT)
	LOCALE_USER_DEFAULT       = MAKELCID(LANG_USER_DEFAULT, SORT_DEFAULT)
	LOCALE_CUSTOM_DEFAULT     = MAKELCID(MAKELANGID(LANG_NEUTRAL, SUBLANG_CUSTOM_DEFAULT), SORT_DEFAULT)
	LOCALE_CUSTOM_UNSPECIFIED = MAKELCID(MAKELANGID(LANG_NEUTRAL, SUBLANG_CUSTOM_UNSPECIFIED), SORT_DEFAULT)
	LOCALE_CUSTOM_UI_DEFAULT  = MAKELCID(MAKELANGID(LANG_NEUTRAL, SUBLANG_UI_CUSTOM_DEFAULT), SORT_DEFAULT)
	LOCALE_NEUTRAL            = MAKELCID(MAKELANGID(LANG_NEUTRAL, SUBLANG_NEUTRAL), SORT_DEFAULT)
	LOCALE_INVARIANT          = MAKELCID(MAKELANGID(LANG_INVARIANT, SUBLANG_NEUTRAL), SORT_DEFAULT)
)

const (
	STATUS_WAIT_0                     = 0x00000000
	STATUS_ABANDONED_WAIT_0           = 0x00000080
	STATUS_USER_APC                   = 0x000000C0
	STATUS_TIMEOUT                    = 0x00000102
	STATUS_PENDING                    = 0x00000103
	DBG_EXCEPTION_HANDLED             = 0x00010001
	DBG_CONTINUE                      = 0x00010002
	STATUS_SEGMENT_NOTIFICATION       = 0x40000005
	DBG_TERMINATE_THREAD              = 0x40010003
	DBG_TERMINATE_PROCESS             = 0x40010004
	DBG_CONTROL_C                     = 0x40010005
	DBG_PRINTEXCEPTION_C              = 0x40010006
	DBG_RIPEXCEPTION                  = 0x40010007
	DBG_CONTROL_BREAK                 = 0x40010008
	DBG_COMMAND_EXCEPTION             = 0x40010009
	STATUS_GUARD_PAGE_VIOLATION       = 0x80000001
	STATUS_DATATYPE_MISALIGNMENT      = 0x80000002
	STATUS_BREAKPOINT                 = 0x80000003
	STATUS_SINGLE_STEP                = 0x80000004
	STATUS_LONGJUMP                   = 0x80000026
	STATUS_UNWIND_CONSOLIDATE         = 0x80000029
	DBG_EXCEPTION_NOT_HANDLED         = 0x80010001
	STATUS_ACCESS_VIOLATION           = 0xC0000005
	STATUS_IN_PAGE_ERROR              = 0xC0000006
	STATUS_INVALID_HANDLE             = 0xC0000008
	STATUS_INVALID_PARAMETER          = 0xC000000D
	STATUS_NO_MEMORY                  = 0xC0000017
	STATUS_ILLEGAL_INSTRUCTION        = 0xC000001D
	STATUS_NONCONTINUABLE_EXCEPTION   = 0xC0000025
	STATUS_INVALID_DISPOSITION        = 0xC0000026
	STATUS_ARRAY_BOUNDS_EXCEEDED      = 0xC000008C
	STATUS_FLOAT_DENORMAL_OPERAND     = 0xC000008D
	STATUS_FLOAT_DIVIDE_BY_ZERO       = 0xC000008E
	STATUS_FLOAT_INEXACT_RESULT       = 0xC000008F
	STATUS_FLOAT_INVALID_OPERATION    = 0xC0000090
	STATUS_FLOAT_OVERFLOW             = 0xC0000091
	STATUS_FLOAT_STACK_CHECK          = 0xC0000092
	STATUS_FLOAT_UNDERFLOW            = 0xC0000093
	STATUS_INTEGER_DIVIDE_BY_ZERO     = 0xC0000094
	STATUS_INTEGER_OVERFLOW           = 0xC0000095
	STATUS_PRIVILEGED_INSTRUCTION     = 0xC0000096
	STATUS_STACK_OVERFLOW             = 0xC00000FD
	STATUS_DLL_NOT_FOUND              = 0xC0000135
	STATUS_ORDINAL_NOT_FOUND          = 0xC0000138
	STATUS_ENTRYPOINT_NOT_FOUND       = 0xC0000139
	STATUS_CONTROL_C_EXIT             = 0xC000013A
	STATUS_DLL_INIT_FAILED            = 0xC0000142
	STATUS_FLOAT_MULTIPLE_FAULTS      = 0xC00002B4
	STATUS_FLOAT_MULTIPLE_TRAPS       = 0xC00002B5
	STATUS_REG_NAT_CONSUMPTION        = 0xC00002C9
	STATUS_STACK_BUFFER_OVERRUN       = 0xC0000409
	STATUS_INVALID_CRUNTIME_PARAMETER = 0xC0000417
	STATUS_ASSERTION_FAILURE          = 0xC0000420
	STATUS_SXS_EARLY_DEACTIVATION     = 0xC015000F
	STATUS_SXS_INVALID_DEACTIVATION   = 0xC0150010
)

const (
	DELETE                   = 0x00010000
	READ_CONTROL             = 0x00020000
	WRITE_DAC                = 0x00040000
	WRITE_OWNER              = 0x00080000
	SYNCHRONIZE              = 0x00100000
	STANDARD_RIGHTS_REQUIRED = 0x000F0000
	STANDARD_RIGHTS_READ     = READ_CONTROL
	STANDARD_RIGHTS_WRITE    = READ_CONTROL
	STANDARD_RIGHTS_EXECUTE  = READ_CONTROL
	STANDARD_RIGHTS_ALL      = 0x001F0000
	SPECIFIC_RIGHTS_ALL      = 0x0000FFFF
	ACCESS_SYSTEM_SECURITY   = 0x01000000
	MAXIMUM_ALLOWED          = 0x02000000
)

const (
	GENERIC_READ    = 0x80000000
	GENERIC_WRITE   = 0x40000000
	GENERIC_EXECUTE = 0x20000000
	GENERIC_ALL     = 0x10000000
)

type LUID_AND_ATTRIBUTES struct {
	Luid       LUID
	Attributes uint32
}

type SID_IDENTIFIER_AUTHORITY struct {
	Value [6]byte
}

type SID struct {}

const (
	SidTypeUser           = 1
	SidTypeGroup          = 2
	SidTypeDomain         = 3
	SidTypeAlias          = 4
	SidTypeWellKnownGroup = 5
	SidTypeDeletedAccount = 6
	SidTypeInvalid        = 7
	SidTypeUnknown        = 8
	SidTypeComputer       = 9
	SidTypeLabel          = 10
)

type SID_AND_ATTRIBUTES struct {
	Sid        *SID
	Attributes uint32
}

const (
	SID_HASH_SIZE = 32
)

type SID_AND_ATTRIBUTES_HASH struct {
	SidCount uint32
	SidAttr  *SID_AND_ATTRIBUTES
	Hash     [SID_HASH_SIZE]uintptr
}

var (
	SECURITY_NULL_SID_AUTHORITY        = SID_IDENTIFIER_AUTHORITY{[6]byte{0, 0, 0, 0, 0, 0}}
	SECURITY_WORLD_SID_AUTHORITY       = SID_IDENTIFIER_AUTHORITY{[6]byte{0, 0, 0, 0, 0, 1}}
	SECURITY_LOCAL_SID_AUTHORITY       = SID_IDENTIFIER_AUTHORITY{[6]byte{0, 0, 0, 0, 0, 2}}
	SECURITY_CREATOR_SID_AUTHORITY     = SID_IDENTIFIER_AUTHORITY{[6]byte{0, 0, 0, 0, 0, 3}}
	SECURITY_NT_AUTHORITY              = SID_IDENTIFIER_AUTHORITY{[6]byte{0, 0, 0, 0, 0, 5}}
	SECURITY_MANDATORY_LABEL_AUTHORITY = SID_IDENTIFIER_AUTHORITY{[6]byte{0, 0, 0, 0, 0, 16}}
)

const (
	SECURITY_NULL_RID          = 0x00000000
	SECURITY_WORLD_RID         = 0x00000000
	SECURITY_LOCAL_RID         = 0x00000000
	SECURITY_LOCAL_LOGON_RID   = 0x00000001
	SECURITY_CREATOR_OWNER_RID = 0x00000000
	SECURITY_CREATOR_GROUP_RID = 0x00000001
)

const (
	SECURITY_DIALUP_RID                 = 0x00000001
	SECURITY_NETWORK_RID                = 0x00000002
	SECURITY_BATCH_RID                  = 0x00000003
	SECURITY_INTERACTIVE_RID            = 0x00000004
	SECURITY_LOGON_IDS_RID              = 0x00000005
	SECURITY_SERVICE_RID                = 0x00000006
	SECURITY_ANONYMOUS_LOGON_RID        = 0x00000007
	SECURITY_PROXY_RID                  = 0x00000008
	SECURITY_ENTERPRISE_CONTROLLERS_RID = 0x00000009
	SECURITY_PRINCIPAL_SELF_RID         = 0x0000000A
	SECURITY_AUTHENTICATED_USER_RID     = 0x0000000B
	SECURITY_RESTRICTED_CODE_RID        = 0x0000000C
	SECURITY_TERMINAL_SERVER_RID        = 0x0000000D
	SECURITY_LOCAL_SYSTEM_RID           = 0x00000012
	SECURITY_LOCAL_SERVICE_RID          = 0x00000013
	SECURITY_NETWORK_SERVICE_RID        = 0x00000014
	SECURITY_NT_NON_UNIQUE              = 0x00000015
	SECURITY_BUILTIN_DOMAIN_RID         = 0x00000020
)

const (
	DOMAIN_USER_RID_ADMIN = 0x000001F4
	DOMAIN_USER_RID_GUEST = 0x000001F5
)

const (
	DOMAIN_GROUP_RID_ADMINS               = 0x00000200
	DOMAIN_GROUP_RID_USERS                = 0x00000201
	DOMAIN_GROUP_RID_GUESTS               = 0x00000202
	DOMAIN_GROUP_RID_COMPUTERS            = 0x00000203
	DOMAIN_GROUP_RID_CONTROLLERS          = 0x00000204
	DOMAIN_GROUP_RID_CERT_ADMINS          = 0x00000205
	DOMAIN_GROUP_RID_SCHEMA_ADMINS        = 0x00000206
	DOMAIN_GROUP_RID_ENTERPRISE_ADMINS    = 0x00000207
	DOMAIN_GROUP_RID_POLICY_ADMINS        = 0x00000208
	DOMAIN_GROUP_RID_READONLY_CONTROLLERS = 0x00000209
)

const (
	DOMAIN_ALIAS_RID_ADMINS                         = 0x00000220
	DOMAIN_ALIAS_RID_USERS                          = 0x00000221
	DOMAIN_ALIAS_RID_GUESTS                         = 0x00000222
	DOMAIN_ALIAS_RID_POWER_USERS                    = 0x00000223
	DOMAIN_ALIAS_RID_ACCOUNT_OPS                    = 0x00000224
	DOMAIN_ALIAS_RID_SYSTEM_OPS                     = 0x00000225
	DOMAIN_ALIAS_RID_PRINT_OPS                      = 0x00000226
	DOMAIN_ALIAS_RID_BACKUP_OPS                     = 0x00000227
	DOMAIN_ALIAS_RID_REPLICATOR                     = 0x00000228
	DOMAIN_ALIAS_RID_RAS_SERVERS                    = 0x00000229
	DOMAIN_ALIAS_RID_PREW2KCOMPACCESS               = 0x0000022A
	DOMAIN_ALIAS_RID_REMOTE_DESKTOP_USERS           = 0x0000022B
	DOMAIN_ALIAS_RID_NETWORK_CONFIGURATION_OPS      = 0x0000022C
	DOMAIN_ALIAS_RID_INCOMING_FOREST_TRUST_BUILDERS = 0x0000022D
	DOMAIN_ALIAS_RID_MONITORING_USERS               = 0x0000022E
	DOMAIN_ALIAS_RID_LOGGING_USERS                  = 0x0000022F
	DOMAIN_ALIAS_RID_AUTHORIZATIONACCESS            = 0x00000230
	DOMAIN_ALIAS_RID_TS_LICENSE_SERVERS             = 0x00000231
	DOMAIN_ALIAS_RID_DCOM_USERS                     = 0x00000232
	DOMAIN_ALIAS_RID_IUSERS                         = 0x00000238
	DOMAIN_ALIAS_RID_CRYPTO_OPERATORS               = 0x00000239
	DOMAIN_ALIAS_RID_CACHEABLE_PRINCIPALS_GROUP     = 0x0000023B
	DOMAIN_ALIAS_RID_NON_CACHEABLE_PRINCIPALS_GROUP = 0x0000023C
	DOMAIN_ALIAS_RID_EVENT_LOG_READERS_GROUP        = 0x0000023D
	DOMAIN_ALIAS_RID_CERTSVC_DCOM_ACCESS_GROUP      = 0x0000023E
)

const (
	SECURITY_MANDATORY_UNTRUSTED_RID         = 0x00000000
	SECURITY_MANDATORY_LOW_RID               = 0x00001000
	SECURITY_MANDATORY_MEDIUM_RID            = 0x00002000
	SECURITY_MANDATORY_MEDIUM_PLUS_RID       = SECURITY_MANDATORY_MEDIUM_RID + 0x00000100
	SECURITY_MANDATORY_HIGH_RID              = 0x00003000
	SECURITY_MANDATORY_SYSTEM_RID            = 0x00004000
	SECURITY_MANDATORY_PROTECTED_PROCESS_RID = 0x00005000
)

const (
	SE_GROUP_MANDATORY          = 0x00000001
	SE_GROUP_ENABLED_BY_DEFAULT = 0x00000002
	SE_GROUP_ENABLED            = 0x00000004
	SE_GROUP_OWNER              = 0x00000008
	SE_GROUP_USE_FOR_DENY_ONLY  = 0x00000010
	SE_GROUP_INTEGRITY          = 0x00000020
	SE_GROUP_INTEGRITY_ENABLED  = 0x00000040
	SE_GROUP_LOGON_ID           = 0xC0000000
	SE_GROUP_RESOURCE           = 0x20000000
)

const (
	ACL_REVISION    = 2
	ACL_REVISION_DS = 4
)

type ACL struct {
	AclRevision byte
	Sbz1        byte
	AclSize     uint16
	AceCount    uint16
	Sbz2        uint16
}

const (
	SE_PRIVILEGE_ENABLED_BY_DEFAULT = 0x00000001
	SE_PRIVILEGE_ENABLED            = 0x00000002
	SE_PRIVILEGE_REMOVED            = 0x00000004
	SE_PRIVILEGE_USED_FOR_ACCESS    = 0x80000000
)

const (
	SE_CREATE_TOKEN_NAME           = "SeCreateTokenPrivilege"
	SE_ASSIGNPRIMARYTOKEN_NAME     = "SeAssignPrimaryTokenPrivilege"
	SE_LOCK_MEMORY_NAME            = "SeLockMemoryPrivilege"
	SE_INCREASE_QUOTA_NAME         = "SeIncreaseQuotaPrivilege"
	SE_UNSOLICITED_INPUT_NAME      = "SeUnsolicitedInputPrivilege"
	SE_MACHINE_ACCOUNT_NAME        = "SeMachineAccountPrivilege"
	SE_TCB_NAME                    = "SeTcbPrivilege"
	SE_SECURITY_NAME               = "SeSecurityPrivilege"
	SE_TAKE_OWNERSHIP_NAME         = "SeTakeOwnershipPrivilege"
	SE_LOAD_DRIVER_NAME            = "SeLoadDriverPrivilege"
	SE_SYSTEM_PROFILE_NAME         = "SeSystemProfilePrivilege"
	SE_SYSTEMTIME_NAME             = "SeSystemtimePrivilege"
	SE_PROF_SINGLE_PROCESS_NAME    = "SeProfileSingleProcessPrivilege"
	SE_INC_BASE_PRIORITY_NAME      = "SeIncreaseBasePriorityPrivilege"
	SE_CREATE_PAGEFILE_NAME        = "SeCreatePagefilePrivilege"
	SE_CREATE_PERMANENT_NAME       = "SeCreatePermanentPrivilege"
	SE_BACKUP_NAME                 = "SeBackupPrivilege"
	SE_RESTORE_NAME                = "SeRestorePrivilege"
	SE_SHUTDOWN_NAME               = "SeShutdownPrivilege"
	SE_DEBUG_NAME                  = "SeDebugPrivilege"
	SE_AUDIT_NAME                  = "SeAuditPrivilege"
	SE_SYSTEM_ENVIRONMENT_NAME     = "SeSystemEnvironmentPrivilege"
	SE_CHANGE_NOTIFY_NAME          = "SeChangeNotifyPrivilege"
	SE_REMOTE_SHUTDOWN_NAME        = "SeRemoteShutdownPrivilege"
	SE_UNDOCK_NAME                 = "SeUndockPrivilege"
	SE_SYNC_AGENT_NAME             = "SeSyncAgentPrivilege"
	SE_ENABLE_DELEGATION_NAME      = "SeEnableDelegationPrivilege"
	SE_MANAGE_VOLUME_NAME          = "SeManageVolumePrivilege"
	SE_IMPERSONATE_NAME            = "SeImpersonatePrivilege"
	SE_CREATE_GLOBAL_NAME          = "SeCreateGlobalPrivilege"
	SE_TRUSTED_CREDMAN_ACCESS_NAME = "SeTrustedCredManAccessPrivilege"
	SE_RELABEL_NAME                = "SeRelabelPrivilege"
	SE_INC_WORKING_SET_NAME        = "SeIncreaseWorkingSetPrivilege"
	SE_TIME_ZONE_NAME              = "SeTimeZonePrivilege"
	SE_CREATE_SYMBOLIC_LINK_NAME   = "SeCreateSymbolicLinkPrivilege"
)

const (
	SecurityAnonymous      = 0
	SecurityIdentification = 1
	SecurityImpersonation  = 2
	SecurityDelegation     = 3
)

const (
	TOKEN_ASSIGN_PRIMARY    = 0x0001
	TOKEN_DUPLICATE         = 0x0002
	TOKEN_IMPERSONATE       = 0x0004
	TOKEN_QUERY             = 0x0008
	TOKEN_QUERY_SOURCE      = 0x0010
	TOKEN_ADJUST_PRIVILEGES = 0x0020
	TOKEN_ADJUST_GROUPS     = 0x0040
	TOKEN_ADJUST_DEFAULT    = 0x0080
	TOKEN_ADJUST_SESSIONID  = 0x0100
	TOKEN_ALL_ACCESS        = STANDARD_RIGHTS_REQUIRED | TOKEN_ASSIGN_PRIMARY | TOKEN_DUPLICATE | TOKEN_IMPERSONATE | TOKEN_QUERY | TOKEN_QUERY_SOURCE | TOKEN_ADJUST_PRIVILEGES | TOKEN_ADJUST_GROUPS | TOKEN_ADJUST_DEFAULT | TOKEN_ADJUST_SESSIONID
	TOKEN_READ              = STANDARD_RIGHTS_READ | TOKEN_QUERY
	TOKEN_WRITE             = STANDARD_RIGHTS_WRITE | TOKEN_ADJUST_PRIVILEGES | TOKEN_ADJUST_GROUPS | TOKEN_ADJUST_DEFAULT
	TOKEN_EXECUTE           = STANDARD_RIGHTS_EXECUTE
)

const (
	TokenPrimary       = 1
	TokenImpersonation = 2
)

const (
	TokenElevationTypeDefault = 1
	TokenElevationTypeFull    = 2
	TokenElevationTypeLimited = 3
)

const (
	TokenUser                  = 1
	TokenGroups                = 2
	TokenPrivileges            = 3
	TokenOwner                 = 4
	TokenPrimaryGroup          = 5
	TokenDefaultDacl           = 6
	TokenSource                = 7
	TokenType                  = 8
	TokenImpersonationLevel    = 9
	TokenStatistics            = 10
	TokenRestrictedSids        = 11
	TokenSessionId             = 12
	TokenGroupsAndPrivileges   = 13
	TokenSessionReference      = 14
	TokenSandBoxInert          = 15
	TokenAuditPolicy           = 16
	TokenOrigin                = 17
	TokenElevationType         = 18
	TokenLinkedToken           = 19
	TokenElevation             = 20
	TokenHasRestrictions       = 21
	TokenAccessInformation     = 22
	TokenVirtualizationAllowed = 23
	TokenVirtualizationEnabled = 24
	TokenIntegrityLevel        = 25
	TokenUIAccess              = 26
	TokenMandatoryPolicy       = 27
	TokenLogonSid              = 28
	MaxTokenInfoClass          = 29
)

type TOKEN_USER struct {
	User SID_AND_ATTRIBUTES
}

type TOKEN_GROUPS struct {
	GroupCount uint32
	Groups     [ANYSIZE_ARRAY]SID_AND_ATTRIBUTES
}

type TOKEN_PRIVILEGES struct {
	PrivilegeCount uint32
	Privileges     [ANYSIZE_ARRAY]LUID_AND_ATTRIBUTES
}

type TOKEN_OWNER struct {
	Owner *SID
}

type TOKEN_PRIMARY_GROUP struct {
	PrimaryGroup *SID
}

type TOKEN_DEFAULT_DACL struct {
	DefaultDacl *ACL
}

type TOKEN_GROUPS_AND_PRIVILEGES struct {
	SidCount            uint32
	SidLength           uint32
	Sids                *SID_AND_ATTRIBUTES
	RestrictedSidCount  uint32
	RestrictedSidLength uint32
	RestrictedSids      *SID_AND_ATTRIBUTES
	PrivilegeCount      uint32
	PrivilegeLength     uint32
	Privileges          *LUID_AND_ATTRIBUTES
	AuthenticationId    LUID
}

type TOKEN_LINKED_TOKEN struct {
	LinkedToken syscall.Handle
}

type TOKEN_ELEVATION struct {
	TokenIsElevated uint32
}

type TOKEN_MANDATORY_LABEL struct {
	Label SID_AND_ATTRIBUTES
}

const (
	TOKEN_MANDATORY_POLICY_OFF             = 0x00000000
	TOKEN_MANDATORY_POLICY_NO_WRITE_UP     = 0x00000001
	TOKEN_MANDATORY_POLICY_NEW_PROCESS_MIN = 0x00000002
	TOKEN_MANDATORY_POLICY_VALID_MASK      = TOKEN_MANDATORY_POLICY_NO_WRITE_UP | TOKEN_MANDATORY_POLICY_NEW_PROCESS_MIN
)

type TOKEN_MANDATORY_POLICY struct {
	Policy uint32
}

type TOKEN_ACCESS_INFORMATION struct {
	SidHash            *SID_AND_ATTRIBUTES_HASH
	RestrictedSidHash  *SID_AND_ATTRIBUTES_HASH
	Privileges         *TOKEN_PRIVILEGES
	AuthenticationId   LUID
	TokenType          int32
	ImpersonationLevel int32
	MandatoryPolicy    TOKEN_MANDATORY_POLICY
	Flags              uint32
}

const (
	TOKEN_SOURCE_LENGTH = 8
)

type TOKEN_SOURCE struct {
	SourceName       [TOKEN_SOURCE_LENGTH]byte
	SourceIdentifier LUID
}

type TOKEN_STATISTICS struct {
	TokenId            LUID
	AuthenticationId   LUID
	ExpirationTime     int64
	TokenType          int32
	ImpersonationLevel int32
	DynamicCharged     uint32
	DynamicAvailable   uint32
	GroupCount         uint32
	PrivilegeCount     uint32
	ModifiedId         LUID
}

type TOKEN_ORIGIN struct {
	OriginatingLogonSession LUID
}

const (
	OWNER_SECURITY_INFORMATION            = 0x00000001
	GROUP_SECURITY_INFORMATION            = 0x00000002
	DACL_SECURITY_INFORMATION             = 0x00000004
	SACL_SECURITY_INFORMATION             = 0x00000008
	LABEL_SECURITY_INFORMATION            = 0x00000010
	PROTECTED_DACL_SECURITY_INFORMATION   = 0x80000000
	PROTECTED_SACL_SECURITY_INFORMATION   = 0x40000000
	UNPROTECTED_DACL_SECURITY_INFORMATION = 0x20000000
	UNPROTECTED_SACL_SECURITY_INFORMATION = 0x10000000
)

const (
	PROCESS_TERMINATE                 = 0x0001
	PROCESS_CREATE_THREAD             = 0x0002
	PROCESS_SET_SESSIONID             = 0x0004
	PROCESS_VM_OPERATION              = 0x0008
	PROCESS_VM_READ                   = 0x0010
	PROCESS_VM_WRITE                  = 0x0020
	PROCESS_DUP_HANDLE                = 0x0040
	PROCESS_CREATE_PROCESS            = 0x0080
	PROCESS_SET_QUOTA                 = 0x0100
	PROCESS_SET_INFORMATION           = 0x0200
	PROCESS_QUERY_INFORMATION         = 0x0400
	PROCESS_SUSPEND_RESUME            = 0x0800
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	PROCESS_ALL_ACCESS                = STANDARD_RIGHTS_REQUIRED | SYNCHRONIZE | 0xFFFF
)

const (
	JOB_OBJECT_ASSIGN_PROCESS          = 0x0001
	JOB_OBJECT_SET_ATTRIBUTES          = 0x0002
	JOB_OBJECT_QUERY                   = 0x0004
	JOB_OBJECT_TERMINATE               = 0x0008
	JOB_OBJECT_SET_SECURITY_ATTRIBUTES = 0x0010
	JOB_OBJECT_ALL_ACCESS              = STANDARD_RIGHTS_REQUIRED | SYNCHRONIZE | 0x001F
)

type IO_COUNTERS struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type JOBOBJECT_BASIC_ACCOUNTING_INFORMATION struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

type JOBOBJECT_BASIC_LIMIT_INFORMATION struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type JOBOBJECT_EXTENDED_LIMIT_INFORMATION struct {
	BasicLimitInformation JOBOBJECT_BASIC_LIMIT_INFORMATION
	IoInfo                IO_COUNTERS
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

type JOBOBJECT_BASIC_PROCESS_ID_LIST struct {
	NumberOfAssignedProcesses uint32
	NumberOfProcessIdsInList  uint32
	ProcessIdList             [1]uintptr
}

type JOBOBJECT_BASIC_UI_RESTRICTIONS struct {
	UIRestrictionsClass uint32
}

type JOBOBJECT_SECURITY_LIMIT_INFORMATION struct {
	SecurityLimitFlags uint32
	JobToken           syscall.Handle
	SidsToDisable      *TOKEN_GROUPS
	PrivilegesToDelete *TOKEN_PRIVILEGES
	RestrictedSids     *TOKEN_GROUPS
}

type JOBOBJECT_END_OF_JOB_TIME_INFORMATION struct {
	EndOfJobTimeAction uint32
}

type JOBOBJECT_ASSOCIATE_COMPLETION_PORT struct {
	CompletionKey  *byte
	CompletionPort syscall.Handle
}

type JOBOBJECT_BASIC_AND_IO_ACCOUNTING_INFORMATION struct {
	BasicInfo JOBOBJECT_BASIC_ACCOUNTING_INFORMATION
	IoInfo    IO_COUNTERS
}

const (
	JOB_OBJECT_TERMINATE_AT_END_OF_JOB = 0
	JOB_OBJECT_POST_AT_END_OF_JOB      = 1
)

const (
	JOB_OBJECT_LIMIT_WORKINGSET                 = 0x00000001
	JOB_OBJECT_LIMIT_PROCESS_TIME               = 0x00000002
	JOB_OBJECT_LIMIT_JOB_TIME                   = 0x00000004
	JOB_OBJECT_LIMIT_ACTIVE_PROCESS             = 0x00000008
	JOB_OBJECT_LIMIT_AFFINITY                   = 0x00000010
	JOB_OBJECT_LIMIT_PRIORITY_CLASS             = 0x00000020
	JOB_OBJECT_LIMIT_PRESERVE_JOB_TIME          = 0x00000040
	JOB_OBJECT_LIMIT_SCHEDULING_CLASS           = 0x00000080
	JOB_OBJECT_LIMIT_PROCESS_MEMORY             = 0x00000100
	JOB_OBJECT_LIMIT_JOB_MEMORY                 = 0x00000200
	JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION = 0x00000400
	JOB_OBJECT_LIMIT_BREAKAWAY_OK               = 0x00000800
	JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK        = 0x00001000
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE          = 0x00002000
	JOB_OBJECT_LIMIT_SUBSET_AFFINITY            = 0x00004000
)

const (
	JOB_OBJECT_UILIMIT_HANDLES          = 0x00000001
	JOB_OBJECT_UILIMIT_READCLIPBOARD    = 0x00000002
	JOB_OBJECT_UILIMIT_WRITECLIPBOARD   = 0x00000004
	JOB_OBJECT_UILIMIT_SYSTEMPARAMETERS = 0x00000008
	JOB_OBJECT_UILIMIT_DISPLAYSETTINGS  = 0x00000010
	JOB_OBJECT_UILIMIT_GLOBALATOMS      = 0x00000020
	JOB_OBJECT_UILIMIT_DESKTOP          = 0x00000040
	JOB_OBJECT_UILIMIT_EXITWINDOWS      = 0x00000080
)

const (
	JOB_OBJECT_SECURITY_NO_ADMIN         = 0x00000001
	JOB_OBJECT_SECURITY_RESTRICTED_TOKEN = 0x00000002
	JOB_OBJECT_SECURITY_ONLY_TOKEN       = 0x00000004
	JOB_OBJECT_SECURITY_FILTER_TOKENS    = 0x00000008
)

const (
	JobObjectBasicAccountingInformation         = 1
	JobObjectBasicLimitInformation              = 2
	JobObjectBasicProcessIdList                 = 3
	JobObjectBasicUIRestrictions                = 4
	JobObjectSecurityLimitInformation           = 5
	JobObjectEndOfJobTimeInformation            = 6
	JobObjectAssociateCompletionPortInformation = 7
	JobObjectBasicAndIoAccountingInformation    = 8
	JobObjectExtendedLimitInformation           = 9
	JobObjectGroupInformation                   = 11
)

const (
	PROCESSOR_INTEL_386     = 386
	PROCESSOR_INTEL_486     = 486
	PROCESSOR_INTEL_PENTIUM = 586
	PROCESSOR_INTEL_IA64    = 2200
	PROCESSOR_AMD_X8664     = 8664
)

const (
	PROCESSOR_ARCHITECTURE_INTEL   = 0
	PROCESSOR_ARCHITECTURE_MIPS    = 1
	PROCESSOR_ARCHITECTURE_ALPHA   = 2
	PROCESSOR_ARCHITECTURE_PPC     = 3
	PROCESSOR_ARCHITECTURE_ARM     = 5
	PROCESSOR_ARCHITECTURE_IA64    = 6
	PROCESSOR_ARCHITECTURE_AMD64   = 9
	PROCESSOR_ARCHITECTURE_UNKNOWN = 0xFFFF
)

const (
	FILE_READ_DATA            = 0x0001
	FILE_LIST_DIRECTORY       = 0x0001
	FILE_WRITE_DATA           = 0x0002
	FILE_ADD_FILE             = 0x0002
	FILE_APPEND_DATA          = 0x0004
	FILE_ADD_SUBDIRECTORY     = 0x0004
	FILE_CREATE_PIPE_INSTANCE = 0x0004
	FILE_READ_EA              = 0x0008
	FILE_WRITE_EA             = 0x0010
	FILE_EXECUTE              = 0x0020
	FILE_TRAVERSE             = 0x0020
	FILE_DELETE_CHILD         = 0x0040
	FILE_READ_ATTRIBUTES      = 0x0080
	FILE_WRITE_ATTRIBUTES     = 0x0100
	FILE_ALL_ACCESS           = STANDARD_RIGHTS_REQUIRED | SYNCHRONIZE | 0x01FF
	FILE_GENERIC_READ         = STANDARD_RIGHTS_READ | FILE_READ_DATA | FILE_READ_ATTRIBUTES | FILE_READ_EA | SYNCHRONIZE
	FILE_GENERIC_WRITE        = STANDARD_RIGHTS_WRITE | FILE_WRITE_DATA | FILE_WRITE_ATTRIBUTES | FILE_WRITE_EA | FILE_APPEND_DATA | SYNCHRONIZE
	FILE_GENERIC_EXECUTE      = STANDARD_RIGHTS_EXECUTE | FILE_READ_ATTRIBUTES | FILE_EXECUTE | SYNCHRONIZE
)

const (
	FILE_SHARE_READ   = 0x00000001
	FILE_SHARE_WRITE  = 0x00000002
	FILE_SHARE_DELETE = 0x00000004
)

const (
	FILE_ATTRIBUTE_READONLY            = 0x00000001
	FILE_ATTRIBUTE_HIDDEN              = 0x00000002
	FILE_ATTRIBUTE_SYSTEM              = 0x00000004
	FILE_ATTRIBUTE_DIRECTORY           = 0x00000010
	FILE_ATTRIBUTE_ARCHIVE             = 0x00000020
	FILE_ATTRIBUTE_DEVICE              = 0x00000040
	FILE_ATTRIBUTE_NORMAL              = 0x00000080
	FILE_ATTRIBUTE_TEMPORARY           = 0x00000100
	FILE_ATTRIBUTE_SPARSE_FILE         = 0x00000200
	FILE_ATTRIBUTE_REPARSE_POINT       = 0x00000400
	FILE_ATTRIBUTE_COMPRESSED          = 0x00000800
	FILE_ATTRIBUTE_OFFLINE             = 0x00001000
	FILE_ATTRIBUTE_NOT_CONTENT_INDEXED = 0x00002000
	FILE_ATTRIBUTE_ENCRYPTED           = 0x00004000
	FILE_ATTRIBUTE_VIRTUAL             = 0x00010000
)

const (
	FILE_CASE_SENSITIVE_SEARCH        = 0x00000001
	FILE_CASE_PRESERVED_NAMES         = 0x00000002
	FILE_UNICODE_ON_DISK              = 0x00000004
	FILE_PERSISTENT_ACLS              = 0x00000008
	FILE_FILE_COMPRESSION             = 0x00000010
	FILE_VOLUME_QUOTAS                = 0x00000020
	FILE_SUPPORTS_SPARSE_FILES        = 0x00000040
	FILE_SUPPORTS_REPARSE_POINTS      = 0x00000080
	FILE_SUPPORTS_REMOTE_STORAGE      = 0x00000100
	FILE_VOLUME_IS_COMPRESSED         = 0x00008000
	FILE_SUPPORTS_OBJECT_IDS          = 0x00010000
	FILE_SUPPORTS_ENCRYPTION          = 0x00020000
	FILE_NAMED_STREAMS                = 0x00040000
	FILE_READ_ONLY_VOLUME             = 0x00080000
	FILE_SEQUENTIAL_WRITE_ONCE        = 0x00100000
	FILE_SUPPORTS_TRANSACTIONS        = 0x00200000
	FILE_SUPPORTS_HARD_LINKS          = 0x00400000
	FILE_SUPPORTS_EXTENDED_ATTRIBUTES = 0x00800000
	FILE_SUPPORTS_OPEN_BY_FILE_ID     = 0x01000000
	FILE_SUPPORTS_USN_JOURNAL         = 0x02000000
)

const (
	MAXIMUM_REPARSE_DATA_BUFFER_SIZE = 16 * 1024
)

const (
	IO_REPARSE_TAG_RESERVED_ZERO  = 0
	IO_REPARSE_TAG_RESERVED_ONE   = 1
	IO_REPARSE_TAG_RESERVED_RANGE = IO_REPARSE_TAG_RESERVED_ONE
)

const (
	IO_REPARSE_TAG_MOUNT_POINT = 0xA0000003
	IO_REPARSE_TAG_HSM         = 0xC0000004
	IO_REPARSE_TAG_HSM2        = 0x80000006
	IO_REPARSE_TAG_SIS         = 0x80000007
	IO_REPARSE_TAG_WIM         = 0x80000008
	IO_REPARSE_TAG_CSV         = 0x80000009
	IO_REPARSE_TAG_DFS         = 0x8000000A
	IO_REPARSE_TAG_SYMLINK     = 0xA000000C
	IO_REPARSE_TAG_DFSR        = 0x80000012
)

type MESSAGE_RESOURCE_ENTRY struct {
	Length uint16
	Flags  uint16
	//Text []byte
}

type MESSAGE_RESOURCE_BLOCK struct {
	LowId           uint32
	HighId          uint32
	OffsetToEntries uint32
}

type MESSAGE_RESOURCE_DATA struct {
	NumberOfBlocks uint32
	//Blocks       []MessageResourceBlock
}

type OSVERSIONINFO struct {
	OSVersionInfoSize uint32
	MajorVersion      uint32
	MinorVersion      uint32
	BuildNumber       uint32
	PlatformId        uint32
	CSDVersion        [128]uint16
}

type OSVERSIONINFOEX struct {
	OSVERSIONINFO
	ServicePackMajor uint16
	ServicePackMinor uint16
	SuiteMask        uint16
	ProductType      uint8
	Reserved         uint8
}

const (
	VER_EQUAL         = 1
	VER_GREATER       = 2
	VER_GREATER_EQUAL = 3
	VER_LESS          = 4
	VER_LESS_EQUAL    = 5
	VER_AND           = 6
	VER_OR            = 7
)

const (
	VER_MINORVERSION     = 0x00000001
	VER_MAJORVERSION     = 0x00000002
	VER_BUILDNUMBER      = 0x00000004
	VER_PLATFORMID       = 0x00000008
	VER_SERVICEPACKMINOR = 0x00000010
	VER_SERVICEPACKMAJOR = 0x00000020
	VER_SUITENAME        = 0x00000040
	VER_PRODUCT_TYPE     = 0x00000080
)

const (
	VER_NT_WORKSTATION       = 0x00000001
	VER_NT_DOMAIN_CONTROLLER = 0x00000002
	VER_NT_SERVER            = 0x00000003
)

const (
	VER_PLATFORM_WIN32s        = 0
	VER_PLATFORM_WIN32_WINDOWS = 1
	VER_PLATFORM_WIN32_NT      = 2
)

type RTL_CRITICAL_SECTION_DEBUG struct {
	Type                      uint16
	CreatorBackTraceIndex     uint16
	CriticalSection           *RTL_CRITICAL_SECTION
	ProcessLocksList          LIST_ENTRY
	EntryCount                uint32
	ContentionCount           uint32
	Flags                     uint32
	CreatorBackTraceIndexHigh uint16
	SpareWORD                 uint16
}

type RTL_CRITICAL_SECTION struct {
	DebugInfo      *RTL_CRITICAL_SECTION_DEBUG
	LockCount      int32
	RecursionCount int32
	OwningThread   syscall.Handle
	LockSemaphore  syscall.Handle
	SpinCount      uintptr
}

const (
	EVENTLOG_SUCCESS          = 0x0000
	EVENTLOG_ERROR_TYPE       = 0x0001
	EVENTLOG_WARNING_TYPE     = 0x0002
	EVENTLOG_INFORMATION_TYPE = 0x0004
	EVENTLOG_AUDIT_SUCCESS    = 0x0008
	EVENTLOG_AUDIT_FAILURE    = 0x0010
)

const (
	KEY_QUERY_VALUE        = 0x0001
	KEY_SET_VALUE          = 0x0002
	KEY_CREATE_SUB_KEY     = 0x0004
	KEY_ENUMERATE_SUB_KEYS = 0x0008
	KEY_NOTIFY             = 0x0010
	KEY_CREATE_LINK        = 0x0020
	KEY_WOW64_32KEY        = 0x0200
	KEY_WOW64_64KEY        = 0x0100
	KEY_READ               = (STANDARD_RIGHTS_READ | KEY_QUERY_VALUE | KEY_ENUMERATE_SUB_KEYS | KEY_NOTIFY) & ^SYNCHRONIZE
	KEY_WRITE              = (STANDARD_RIGHTS_WRITE | KEY_SET_VALUE | KEY_CREATE_SUB_KEY) & ^SYNCHRONIZE
	KEY_EXECUTE            = KEY_READ & ^SYNCHRONIZE
	KEY_ALL_ACCESS         = (STANDARD_RIGHTS_ALL | KEY_QUERY_VALUE | KEY_SET_VALUE | KEY_CREATE_SUB_KEY | KEY_ENUMERATE_SUB_KEYS | KEY_NOTIFY | KEY_CREATE_LINK) & ^SYNCHRONIZE
)

const (
	REG_OPTION_NON_VOLATILE   = 0x00000000
	REG_OPTION_VOLATILE       = 0x00000001
	REG_OPTION_CREATE_LINK    = 0x00000002
	REG_OPTION_BACKUP_RESTORE = 0x00000004
)

const (
	REG_CREATED_NEW_KEY     = 0x00000001
	REG_OPENED_EXISTING_KEY = 0x00000002
)

const (
	REG_NONE                = 0
	REG_SZ                  = 1
	REG_EXPAND_SZ           = 2
	REG_BINARY              = 3
	REG_DWORD               = 4
	REG_DWORD_LITTLE_ENDIAN = 4
	REG_DWORD_BIG_ENDIAN    = 5
	REG_LINK                = 6
	REG_MULTI_SZ            = 7
	REG_QWORD               = 11
	REG_QWORD_LITTLE_ENDIAN = 11
)

const (
	SERVICE_KERNEL_DRIVER       = 0x00000001
	SERVICE_FILE_SYSTEM_DRIVER  = 0x00000002
	SERVICE_ADAPTER             = 0x00000004
	SERVICE_RECOGNIZER_DRIVER   = 0x00000008
	SERVICE_DRIVER              = SERVICE_KERNEL_DRIVER | SERVICE_FILE_SYSTEM_DRIVER | SERVICE_RECOGNIZER_DRIVER
	SERVICE_WIN32_OWN_PROCESS   = 0x00000010
	SERVICE_WIN32_SHARE_PROCESS = 0x00000020
	SERVICE_WIN32               = SERVICE_WIN32_OWN_PROCESS | SERVICE_WIN32_SHARE_PROCESS
	SERVICE_INTERACTIVE_PROCESS = 0x00000100
)

const (
	SERVICE_BOOT_START   = 0x00000000
	SERVICE_SYSTEM_START = 0x00000001
	SERVICE_AUTO_START   = 0x00000002
	SERVICE_DEMAND_START = 0x00000003
	SERVICE_DISABLED     = 0x00000004
)

const (
	SERVICE_ERROR_IGNORE   = 0x00000000
	SERVICE_ERROR_NORMAL   = 0x00000001
	SERVICE_ERROR_SEVERE   = 0x00000002
	SERVICE_ERROR_CRITICAL = 0x00000003
)

var (
	procRtlMoveMemory       = modkernel32.NewProc("RtlMoveMemory")
	procRtlZeroMemory       = modkernel32.NewProc("RtlZeroMemory")
	procVerSetConditionMask = modkernel32.NewProc("VerSetConditionMask")
)

func RtlMoveMemory(destination *byte, source *byte, length uintptr) {
	syscall.Syscall(
		procRtlMoveMemory.Addr(),
		3,
		uintptr(unsafe.Pointer(destination)),
		uintptr(unsafe.Pointer(source)),
		length)
}

func RtlZeroMemory(destination *byte, length uintptr) {
	syscall.Syscall(
		procRtlZeroMemory.Addr(),
		2,
		uintptr(unsafe.Pointer(destination)),
		length,
		0)
}

func VerSetConditionMask(conditionMask uint64, typeBitMask uint32, condition uint8) uint64 {
	r1, _, _ := syscall.Syscall(
		procVerSetConditionMask.Addr(),
		3,
		uintptr(conditionMask),
		uintptr(typeBitMask),
		uintptr(condition))
	return uint64(r1)
}
