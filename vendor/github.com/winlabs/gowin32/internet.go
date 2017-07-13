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
	"unsafe"
)

type InternetOpenType uint32

const (
	InternetOpenDirect                   InternetOpenType = wrappers.INTERNET_OPEN_TYPE_DIRECT
	InternetOpenPreconfig                InternetOpenType = wrappers.INTERNET_OPEN_TYPE_PRECONFIG
	InternetOpenPreconfigWithNoAutoproxy InternetOpenType = wrappers.INTERNET_OPEN_TYPE_PRECONFIG_WITH_NO_AUTOPROXY
)

type InternetOpenFlags uint32

const (
	InternetOpenAsync     InternetOpenFlags = wrappers.INTERNET_FLAG_ASYNC
	InternetOpenFromCache InternetOpenFlags = wrappers.INTERNET_FLAG_FROM_CACHE
	InternetOpenOffline   InternetOpenFlags = wrappers.INTERNET_FLAG_OFFLINE
)

type InternetService uint32

const (
	InternetServiceFTP  InternetService = wrappers.INTERNET_SERVICE_FTP
	InternetServiceHTTP InternetService = wrappers.INTERNET_SERVICE_HTTP
)

type HTTPRequestFlags uint32

const (
	HTTPRequestCacheIfNetFail        HTTPRequestFlags = wrappers.INTERNET_FLAG_CACHE_IF_NET_FAIL
	HTTPRequestHyperlink             HTTPRequestFlags = wrappers.INTERNET_FLAG_HYPERLINK
	HTTPRequestIgnoreCertCNInvalid   HTTPRequestFlags = wrappers.INTERNET_FLAG_IGNORE_CERT_CN_INVALID
	HTTPRequestIgnoreCertDateInvalid HTTPRequestFlags = wrappers.INTERNET_FLAG_IGNORE_CERT_DATE_INVALID
	HTTPRequestIgnoreRedirectToHTTP  HTTPRequestFlags = wrappers.INTERNET_FLAG_IGNORE_REDIRECT_TO_HTTP
	HTTPRequestIgnoreRedirectToHTTPS HTTPRequestFlags = wrappers.INTERNET_FLAG_IGNORE_REDIRECT_TO_HTTPS
	HTTPRequestKeepConnection        HTTPRequestFlags = wrappers.INTERNET_FLAG_KEEP_CONNECTION
	HTTPRequestNeedFile              HTTPRequestFlags = wrappers.INTERNET_FLAG_NEED_FILE
	HTTPRequestNoAuth                HTTPRequestFlags = wrappers.INTERNET_FLAG_NO_AUTH
	HTTPRequestNoAutoRedirect        HTTPRequestFlags = wrappers.INTERNET_FLAG_NO_AUTO_REDIRECT
	HTTPRequestNoCacheWrite          HTTPRequestFlags = wrappers.INTERNET_FLAG_NO_CACHE_WRITE
	HTTPRequestNoCookies             HTTPRequestFlags = wrappers.INTERNET_FLAG_NO_COOKIES
	HTTPRequestNoUI                  HTTPRequestFlags = wrappers.INTERNET_FLAG_NO_UI
	HTTPRequestPragmaNoCache         HTTPRequestFlags = wrappers.INTERNET_FLAG_PRAGMA_NOCACHE
	HTTPRequestReload                HTTPRequestFlags = wrappers.INTERNET_FLAG_RELOAD
	HTTPRequestResynchronize         HTTPRequestFlags = wrappers.INTERNET_FLAG_RESYNCHRONIZE
	HTTPRequestSecure                HTTPRequestFlags = wrappers.INTERNET_FLAG_SECURE
)

type HTTPStatusCode uint32

const (
	HTTPStatusContinue            HTTPStatusCode = wrappers.HTTP_STATUS_CONTINUE
	HTTPStatusSwitchProtocols     HTTPStatusCode = wrappers.HTTP_STATUS_SWITCH_PROTOCOLS
	HTTPStatusOK                  HTTPStatusCode = wrappers.HTTP_STATUS_OK
	HTTPStatusCreated             HTTPStatusCode = wrappers.HTTP_STATUS_CREATED
	HTTPStatusAccepted            HTTPStatusCode = wrappers.HTTP_STATUS_ACCEPTED
	HTTPStatusPartial             HTTPStatusCode = wrappers.HTTP_STATUS_PARTIAL
	HTTPStatusNoContent           HTTPStatusCode = wrappers.HTTP_STATUS_NO_CONTENT
	HTTPStatusResetContent        HTTPStatusCode = wrappers.HTTP_STATUS_RESET_CONTENT
	HTTPStatusPartialContent      HTTPStatusCode = wrappers.HTTP_STATUS_PARTIAL_CONTENT
	HTTPStatusAmbiguous           HTTPStatusCode = wrappers.HTTP_STATUS_AMBIGUOUS
	HTTPStatusMoved               HTTPStatusCode = wrappers.HTTP_STATUS_MOVED
	HTTPStatusRedirect            HTTPStatusCode = wrappers.HTTP_STATUS_REDIRECT
	HTTPStatusRedirectMethod      HTTPStatusCode = wrappers.HTTP_STATUS_REDIRECT_METHOD
	HTTPStatusNotModified         HTTPStatusCode = wrappers.HTTP_STATUS_NOT_MODIFIED
	HTTPStatusUseProxy            HTTPStatusCode = wrappers.HTTP_STATUS_USE_PROXY
	HTTPStatusRedirectKeepVerb    HTTPStatusCode = wrappers.HTTP_STATUS_REDIRECT_KEEP_VERB
	HTTPStatusBadRequest          HTTPStatusCode = wrappers.HTTP_STATUS_BAD_REQUEST
	HTTPStatusDenied              HTTPStatusCode = wrappers.HTTP_STATUS_DENIED
	HTTPStatusPaymentRequired     HTTPStatusCode = wrappers.HTTP_STATUS_PAYMENT_REQ
	HTTPStatusForbidden           HTTPStatusCode = wrappers.HTTP_STATUS_FORBIDDEN
	HTTPStatusNotFound            HTTPStatusCode = wrappers.HTTP_STATUS_NOT_FOUND
	HTTPStatusBadMethod           HTTPStatusCode = wrappers.HTTP_STATUS_BAD_METHOD
	HTTPStatusNoneAcceptable      HTTPStatusCode = wrappers.HTTP_STATUS_NONE_ACCEPTABLE
	HTTPStatusProxyAuthRequired   HTTPStatusCode = wrappers.HTTP_STATUS_PROXY_AUTH_REQ
	HTTPStatusRequestTimeout      HTTPStatusCode = wrappers.HTTP_STATUS_REQUEST_TIMEOUT
	HTTPStatusConflict            HTTPStatusCode = wrappers.HTTP_STATUS_CONFLICT
	HTTPStatusGone                HTTPStatusCode = wrappers.HTTP_STATUS_GONE
	HTTPStatusLengthRequired      HTTPStatusCode = wrappers.HTTP_STATUS_LENGTH_REQUIRED
	HTTPStatusPreconditionFailed  HTTPStatusCode = wrappers.HTTP_STATUS_PRECOND_FAILED
	HTTPStatusRequestTooLarge     HTTPStatusCode = wrappers.HTTP_STATUS_REQUEST_TOO_LARGE
	HTTPStatusURITooLong          HTTPStatusCode = wrappers.HTTP_STATUS_URI_TOO_LONG
	HTTPStatusUnsupportedMedia    HTTPStatusCode = wrappers.HTTP_STATUS_UNSUPPORTED_MEDIA
	HTTPStatusRetryWith           HTTPStatusCode = wrappers.HTTP_STATUS_RETRY_WITH
	HTTPStatusServerError         HTTPStatusCode = wrappers.HTTP_STATUS_SERVER_ERROR
	HTTPStatusNotSupported        HTTPStatusCode = wrappers.HTTP_STATUS_NOT_SUPPORTED
	HTTPStatusBadGateway          HTTPStatusCode = wrappers.HTTP_STATUS_BAD_GATEWAY
	HTTPStatusServiceUnavailable  HTTPStatusCode = wrappers.HTTP_STATUS_SERVICE_UNAVAIL
	HTTPStatusGatewayTimeout      HTTPStatusCode = wrappers.HTTP_STATUS_GATEWAY_TIMEOUT
	HTTPStatusVersionNotSupported HTTPStatusCode = wrappers.HTTP_STATUS_VERSION_NOT_SUP
)

type URLRequestFlags uint32

const (
	URLRequestExistingConnect       URLRequestFlags = wrappers.INTERNET_FLAG_EXISTING_CONNECT
	URLRequestHyperlink             URLRequestFlags = wrappers.INTERNET_FLAG_HYPERLINK
	URLRequestIgnoreCertCNInvalid   URLRequestFlags = wrappers.INTERNET_FLAG_IGNORE_CERT_CN_INVALID
	URLRequestIgnoreCertDateInvalid URLRequestFlags = wrappers.INTERNET_FLAG_IGNORE_CERT_DATE_INVALID
	URLRequestIgnoreRedirectToHTTP  URLRequestFlags = wrappers.INTERNET_FLAG_IGNORE_REDIRECT_TO_HTTP
	URLRequestIgnoreRedirectToHTTPS URLRequestFlags = wrappers.INTERNET_FLAG_IGNORE_REDIRECT_TO_HTTPS
	URLRequestKeepConnection        URLRequestFlags = wrappers.INTERNET_FLAG_KEEP_CONNECTION
	URLRequestNeedFile              URLRequestFlags = wrappers.INTERNET_FLAG_NEED_FILE
	URLRequestNoAuth                URLRequestFlags = wrappers.INTERNET_FLAG_NO_AUTH
	URLRequestNoAutoRedirect        URLRequestFlags = wrappers.INTERNET_FLAG_NO_AUTO_REDIRECT
	URLRequestNoCacheWrite          URLRequestFlags = wrappers.INTERNET_FLAG_NO_CACHE_WRITE
	URLRequestNoCookies             URLRequestFlags = wrappers.INTERNET_FLAG_NO_COOKIES
	URLRequestNoUI                  URLRequestFlags = wrappers.INTERNET_FLAG_NO_UI
	URLRequestPassive               URLRequestFlags = wrappers.INTERNET_FLAG_PASSIVE
	URLRequestPragmaNoCache         URLRequestFlags = wrappers.INTERNET_FLAG_PRAGMA_NOCACHE
	URLRequestRawData               URLRequestFlags = wrappers.INTERNET_FLAG_RAW_DATA
	URLRequestReload                URLRequestFlags = wrappers.INTERNET_FLAG_RELOAD
	URLRequestResynchronize         URLRequestFlags = wrappers.INTERNET_FLAG_RESYNCHRONIZE
	URLRequestSecure                URLRequestFlags = wrappers.INTERNET_FLAG_SECURE
)

type InternetObject struct {
	handle syscall.Handle
}

func (self *InternetObject) Close() error {
	if self.handle != 0 {
		if err := wrappers.InternetCloseHandle(self.handle); err != nil {
			return NewWindowsError("InternetCloseHandle", err)
		}
		self.handle = 0
	}
	return nil
}

type InternetFile struct {
	InternetObject
}

func (self *InternetFile) GetBytesAvailable() (int, error) {
	var bytesAvailable uint32
	if err := wrappers.InternetQueryDataAvailable(self.handle, &bytesAvailable, 0, 0); err != nil {
		return 0, NewWindowsError("InternetQueryDataAvailable", err)
	}
	return int(bytesAvailable), nil
}

func (self *InternetFile) Read(p []byte) (int, error) {
	var bytesRead uint32
	if err := wrappers.InternetReadFile(self.handle, &p[0], uint32(len(p)), &bytesRead); err != nil {
		return 0, NewWindowsError("InternetReadFile", err)
	}
	return int(bytesRead), nil
}

type HTTPRequest struct {
	InternetFile
}

func (self *HTTPRequest) GetStatusCode() (HTTPStatusCode, error) {
	var statusCode uint32
	bufferLength := uint32(unsafe.Sizeof(statusCode))
	index := uint32(0)
	err := wrappers.HttpQueryInfo(
		self.handle,
		wrappers.HTTP_QUERY_STATUS_CODE | wrappers.HTTP_QUERY_FLAG_NUMBER,
		(*byte)(unsafe.Pointer(&statusCode)),
		&bufferLength,
		&index)
	if err != nil {
		return 0, NewWindowsError("HttpQueryInfo", err)
	}
	return HTTPStatusCode(statusCode), nil
}

func (self *HTTPRequest) Send(headers string, optional []byte) error {
	var headersRaw *uint16
	if headers != "" {
		headersRaw = syscall.StringToUTF16Ptr(headers)
	}
	var optionalRaw *byte
	var optionalLength uint32
	if optional != nil {
		optionalRaw = &optional[0]
		optionalLength = uint32(len(optional))
	}
	err := wrappers.HttpSendRequest(
		self.handle,
		headersRaw,
		uint32(len(headers)),
		optionalRaw,
		optionalLength)
	if err != nil {
		return NewWindowsError("HttpSendRequest", err)
	}
	return nil
}

type InternetConnection struct {
	InternetObject
}

func (self *InternetConnection) OpenHTTPRequest(verb string, objectName string, version string, referer string, acceptTypes []string, flags HTTPRequestFlags) (*HTTPRequest, error) {
	var versionRaw *uint16
	if version != "" {
		versionRaw = syscall.StringToUTF16Ptr(version)
	}
	var refererRaw *uint16
	if referer != "" {
		refererRaw = syscall.StringToUTF16Ptr(referer)
	}
	var acceptTypesRaw **uint16
	if acceptTypes != nil {
		acceptTypesPtrs := make([]*uint16, len(acceptTypes), len(acceptTypes) + 1)
		for i := range acceptTypes {
			acceptTypesPtrs[i] = syscall.StringToUTF16Ptr(acceptTypes[i])
		}
		acceptTypesPtrs = append(acceptTypesPtrs, nil)
		acceptTypesRaw = &acceptTypesPtrs[0]
	}
	handle, err := wrappers.HttpOpenRequest(
		self.handle,
		syscall.StringToUTF16Ptr(verb),
		syscall.StringToUTF16Ptr(objectName),
		versionRaw,
		refererRaw,
		acceptTypesRaw,
		uint32(flags),
		0)
	if err != nil {
		return nil, NewWindowsError("HttpOpenRequest", err)
	}
	return &HTTPRequest{InternetFile{InternetObject{handle: handle}}}, nil
}

type InternetSession struct {
	InternetObject
}

func OpenInternetSession(agent string, openType InternetOpenType, flags InternetOpenFlags) (*InternetSession, error) {
	handle, err := wrappers.InternetOpen(
		syscall.StringToUTF16Ptr(agent),
		uint32(openType),
		nil,
		nil,
		uint32(flags))
	if err != nil {
		return nil, NewWindowsError("InternetOpen", err)
	}
	return &InternetSession{InternetObject{handle: handle}}, nil
}

func (self *InternetSession) Connect(serverName string, serverPort uint, service InternetService) (*InternetConnection, error) {
	handle, err := wrappers.InternetConnect(
		self.handle,
		syscall.StringToUTF16Ptr(serverName),
		uint16(serverPort),
		nil,
		nil,
		uint32(service),
		0,
		0)
	if err != nil {
		return nil, NewWindowsError("InternetConnect", err)
	}
	return &InternetConnection{InternetObject{handle: handle}}, nil
}

func (self *InternetSession) OpenURL(url string, headers string, flags URLRequestFlags) (*InternetFile, error) {
	var headersRaw *uint16
	if headers != "" {
		headersRaw = syscall.StringToUTF16Ptr(headers)
	}
	handle, err := wrappers.InternetOpenUrl(
		self.handle,
		syscall.StringToUTF16Ptr(url),
		headersRaw,
		uint32(len(headers)),
		uint32(flags),
		0)
	if err != nil {
		return nil, NewWindowsError("InternetOpenUrl", err)
	}
	return &InternetFile{InternetObject{handle: handle}}, nil
}
