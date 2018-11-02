package damage

// Language is an Apple UDIF Languageuage code
type Language int

// etc., see https://github.com/phracker/MacOSX-SDKs/blob/master/MacOSX10.6.sdk/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/CarbonCore.framework/Versions/A/Headers/Script.h
const (
	// LanguageEnglish represents the English Languageuage, as if there was a single one.
	LanguageEnglish           = 0
	LanguageFrench            = 1
	LanguageGerman            = 2
	LanguageItalian           = 3
	LanguageDutch             = 4
	LanguageSwedish           = 5
	LanguageSpanish           = 6
	LanguageDanish            = 7
	LanguagePortuguese        = 8
	LanguageNorwegian         = 9
	LanguageHebrew            = 10
	LanguageJapanese          = 11
	LanguageArabic            = 12
	LanguageFinnish           = 13
	LanguageGreek             = 14
	LanguageIcelandic         = 15
	LanguageMaltese           = 16
	LanguageTurkish           = 17
	LanguageCroatian          = 18
	LanguageTradChinese       = 19
	LanguageUrdu              = 20
	LanguageHindi             = 21
	LanguageThai              = 22
	LanguageKorean            = 23
	LanguageLithuanian        = 24
	LanguagePolish            = 25
	LanguageHungarian         = 26
	LanguageEstonian          = 27
	LanguageLatvian           = 28
	LanguageSami              = 29
	LanguageFaroese           = 30
	LanguageFarsi             = 31
	LanguageRussian           = 32
	LanguageSimpChinese       = 33
	LanguageFlemish           = 34
	LanguageIrishGaelic       = 35
	LanguageAlbanian          = 36
	LanguageRomanian          = 37
	LanguageCzech             = 38
	LanguageSlovak            = 39
	LanguageSlovenian         = 40
	LanguageYiddish           = 41
	LanguageSerbian           = 42
	LanguageMacedonian        = 43
	LanguageBulgarian         = 44
	LanguageUkrainian         = 45
	LanguageByelorussian      = 46
	LanguageUzbek             = 47
	LanguageKazakh            = 48
	LanguageAzerbaijani       = 49
	LanguageAzerbaijanAr      = 50
	LanguageArmenian          = 51
	LanguageGeorgian          = 52
	LanguageMoldavian         = 53
	LanguageKirghiz           = 54
	LanguageTajiki            = 55
	LanguageTurkmen           = 56
	LanguageMongolian         = 57
	LanguageMongolianCyr      = 58
	LanguagePashto            = 59
	LanguageKurdish           = 60
	LanguageKashmiri          = 61
	LanguageSindhi            = 62
	LanguageTibetan           = 63
	LanguageNepali            = 64
	LanguageSanskrit          = 65
	LanguageMarathi           = 66
	LanguageBengali           = 67
	LanguageAssamese          = 68
	LanguageGujarati          = 69
	LanguagePunjabi           = 70
	LanguageOriya             = 71
	LanguageMalayalam         = 72
	LanguageKannada           = 73
	LanguageTamil             = 74
	LanguageTelugu            = 75
	LanguageSinhalese         = 76
	LanguageBurmese           = 77
	LanguageKhmer             = 78
	LanguageLao               = 79
	LanguageVietnamese        = 80
	LanguageIndonesian        = 81
	LanguageTagalog           = 82
	LanguageMalayRoman        = 83
	LanguageMalayArabic       = 84
	LanguageAmharic           = 85
	LanguageTigrinya          = 86
	LanguageOromo             = 87
	LanguageSomali            = 88
	LanguageSwahili           = 89
	LanguageRuanda            = 90
	LanguageRundi             = 91
	LanguageChewa             = 92
	LanguageMalagasy          = 93
	LanguageEsperanto         = 94
	LanguageWelsh             = 128
	LanguageBasque            = 129
	LanguageCatalan           = 130
	LanguageLatin             = 131
	LanguageQuechua           = 132
	LanguageGuarani           = 133
	LanguageAymara            = 134
	LanguageTatar             = 135
	LanguageUighur            = 136
	LanguageDzongkha          = 137
	LanguageJavaneseRom       = 138
	LanguageSundaneseRom      = 139
	LanguageGalician          = 140
	LanguageAfrikaans         = 141
	LanguageBreton            = 142
	LanguageInuktitut         = 143
	LanguageScottishGaelic    = 144
	LanguageManxGaelic        = 145
	LanguageIrishGaelicScript = 146
	LanguageTongan            = 147
	LanguageGreekAncient      = 148
	LanguageGreenlandic       = 149
	LanguageAzerbaijanRoman   = 150
	LanguageNynorsk           = 151
	LanguageUnspecified       = 32767
)

func (l Language) String() string {
	switch l {
	case LanguageEnglish:
		return "English"
	case LanguageFrench:
		return "French"
	case LanguageGerman:
		return "German"
	case LanguageItalian:
		return "Italian"
	case LanguageDutch:
		return "Dutch"
	case LanguageSwedish:
		return "Swedish"
	case LanguageSpanish:
		return "Spanish"
	case LanguageDanish:
		return "Danish"
	case LanguagePortuguese:
		return "Portuguese"
	case LanguageNorwegian:
		return "Norwegian"
	case LanguageHebrew:
		return "Hebrew"
	case LanguageJapanese:
		return "Japanese"
	case LanguageArabic:
		return "Arabic"
	case LanguageFinnish:
		return "Finnish"
	case LanguageGreek:
		return "Greek"
	case LanguageIcelandic:
		return "Icelandic"
	case LanguageMaltese:
		return "Maltese"
	case LanguageTurkish:
		return "Turkish"
	case LanguageCroatian:
		return "Croatian"
	case LanguageTradChinese:
		return "TradChinese"
	case LanguageUrdu:
		return "Urdu"
	case LanguageHindi:
		return "Hindi"
	case LanguageThai:
		return "Thai"
	case LanguageKorean:
		return "Korean"
	case LanguageLithuanian:
		return "Lithuanian"
	case LanguagePolish:
		return "Polish"
	case LanguageHungarian:
		return "Hungarian"
	case LanguageEstonian:
		return "Estonian"
	case LanguageLatvian:
		return "Latvian"
	case LanguageSami:
		return "Sami"
	case LanguageFaroese:
		return "Faroese"
	case LanguageFarsi:
		return "Farsi"
	case LanguageRussian:
		return "Russian"
	case LanguageSimpChinese:
		return "SimpChinese"
	case LanguageFlemish:
		return "Flemish"
	case LanguageIrishGaelic:
		return "IrishGaelic"
	case LanguageAlbanian:
		return "Albanian"
	case LanguageRomanian:
		return "Romanian"
	case LanguageCzech:
		return "Czech"
	case LanguageSlovak:
		return "Slovak"
	case LanguageSlovenian:
		return "Slovenian"
	case LanguageYiddish:
		return "Yiddish"
	case LanguageSerbian:
		return "Serbian"
	case LanguageMacedonian:
		return "Macedonian"
	case LanguageBulgarian:
		return "Bulgarian"
	case LanguageUkrainian:
		return "Ukrainian"
	case LanguageByelorussian:
		return "Byelorussian"
	case LanguageUzbek:
		return "Uzbek"
	case LanguageKazakh:
		return "Kazakh"
	case LanguageAzerbaijani:
		return "Azerbaijani"
	case LanguageAzerbaijanAr:
		return "AzerbaijanAr"
	case LanguageArmenian:
		return "Armenian"
	case LanguageGeorgian:
		return "Georgian"
	case LanguageMoldavian:
		return "Moldavian"
	case LanguageKirghiz:
		return "Kirghiz"
	case LanguageTajiki:
		return "Tajiki"
	case LanguageTurkmen:
		return "Turkmen"
	case LanguageMongolian:
		return "Mongolian"
	case LanguageMongolianCyr:
		return "MongolianCyr"
	case LanguagePashto:
		return "Pashto"
	case LanguageKurdish:
		return "Kurdish"
	case LanguageKashmiri:
		return "Kashmiri"
	case LanguageSindhi:
		return "Sindhi"
	case LanguageTibetan:
		return "Tibetan"
	case LanguageNepali:
		return "Nepali"
	case LanguageSanskrit:
		return "Sanskrit"
	case LanguageMarathi:
		return "Marathi"
	case LanguageBengali:
		return "Bengali"
	case LanguageAssamese:
		return "Assamese"
	case LanguageGujarati:
		return "Gujarati"
	case LanguagePunjabi:
		return "Punjabi"
	case LanguageOriya:
		return "Oriya"
	case LanguageMalayalam:
		return "Malayalam"
	case LanguageKannada:
		return "Kannada"
	case LanguageTamil:
		return "Tamil"
	case LanguageTelugu:
		return "Telugu"
	case LanguageSinhalese:
		return "Sinhalese"
	case LanguageBurmese:
		return "Burmese"
	case LanguageKhmer:
		return "Khmer"
	case LanguageLao:
		return "Lao"
	case LanguageVietnamese:
		return "Vietnamese"
	case LanguageIndonesian:
		return "Indonesian"
	case LanguageTagalog:
		return "Tagalog"
	case LanguageMalayRoman:
		return "MalayRoman"
	case LanguageMalayArabic:
		return "MalayArabic"
	case LanguageAmharic:
		return "Amharic"
	case LanguageTigrinya:
		return "Tigrinya"
	case LanguageOromo:
		return "Oromo"
	case LanguageSomali:
		return "Somali"
	case LanguageSwahili:
		return "Swahili"
	case LanguageRuanda:
		return "Ruanda"
	case LanguageRundi:
		return "Rundi"
	case LanguageChewa:
		return "Chewa"
	case LanguageMalagasy:
		return "Malagasy"
	case LanguageEsperanto:
		return "Esperanto"
	case LanguageWelsh:
		return "Welsh"
	case LanguageBasque:
		return "Basque"
	case LanguageCatalan:
		return "Catalan"
	case LanguageLatin:
		return "Latin"
	case LanguageQuechua:
		return "Quechua"
	case LanguageGuarani:
		return "Guarani"
	case LanguageAymara:
		return "Aymara"
	case LanguageTatar:
		return "Tatar"
	case LanguageUighur:
		return "Uighur"
	case LanguageDzongkha:
		return "Dzongkha"
	case LanguageJavaneseRom:
		return "JavaneseRom"
	case LanguageSundaneseRom:
		return "SundaneseRom"
	case LanguageGalician:
		return "Galician"
	case LanguageAfrikaans:
		return "Afrikaans"
	case LanguageBreton:
		return "Breton"
	case LanguageInuktitut:
		return "Inuktitut"
	case LanguageScottishGaelic:
		return "ScottishGaelic"
	case LanguageManxGaelic:
		return "ManxGaelic"
	case LanguageIrishGaelicScript:
		return "IrishGaelicScript"
	case LanguageTongan:
		return "Tongan"
	case LanguageGreekAncient:
		return "GreekAncient"
	case LanguageGreenlandic:
		return "Greenlandic"
	case LanguageAzerbaijanRoman:
		return "AzerbaijanRoman"
	case LanguageNynorsk:
		return "Nynorsk"
	case LanguageUnspecified:
		return "Unspecified"
	default:
		return "Invalid"
	}
}
