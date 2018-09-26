package native

// These DLLs are known to ship in "all" versions of
// Windows without needing any particular component installed
// cf. https://msdn.microsoft.com/en-us/library/ee391643(v=vs.85).aspx
// (by the MSDN law, this link will be dead by the time you next need it.)
// (so, the article name is "DLLs Included with Server Core")
var knownBuiltinDLLs = map[string]string{
	// Windows System Services
	"clfsw32.dll":  "Log file management",
	"dbghelp.dll":  "Debugging helper",
	"dciman32.dll": "Graphics support",
	"fltlib.dll":   "Minifilter management",
	"gdi32.dll":    "Graphics support",
	"kernel32.dll": "Windows system kernel",
	"ntdll.dll":    "Windows internal",
	"ole32.dll":    "Object management",
	"oleaut32.dll": "Object management",
	"psapi.dll":    "Performance monitoring",
	"user32.dll":   "User objects",
	"userenv.dll":  "User profile support",

	// Authentication security
	"advapi32.dll": "HTTP authentication and credential management",
	"crypt32.dll":  "Authentication security",
	"cryptdll.dll": "Authentication security",
	"cryptnet.dll": "Secure channel (X.509 certificates)",
	"cryptui.dll":  "Credential management",
	"netapi32.dll": "Authentication security",
	"secur32.dll":  "Authentication security",
	"wintrust.dll": "Catalog functions",

	// User interface
	"mlang.dll":   "Multiple language support",
	"msctf.dll":   "Text Services Framework (TSF)",
	"shell32.dll": "User Interface",
	"shlwapi.dll": "User Interface",
}
