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

	"fmt"
)

const (
	ErrorFileNotFound        = wrappers.ERROR_FILE_NOT_FOUND
	ErrorAccessDenied        = wrappers.ERROR_ACCESS_DENIED
	ErrorGeneralFailure      = wrappers.ERROR_GEN_FAILURE
	ErrorSharingViolation    = wrappers.ERROR_SHARING_VIOLATION
	ErrorInvalidParameter    = wrappers.ERROR_INVALID_PARAMETER
	ErrorBrokenPipe          = wrappers.ERROR_BROKEN_PIPE
	ErrorServiceDoesNotExist = wrappers.ERROR_SERVICE_DOES_NOT_EXIST
)

type WindowsError struct {
	functionName string
	innerError   error
}

func NewWindowsError(functionName string, innerError error) *WindowsError {
	return &WindowsError{functionName, innerError}
}

func (self *WindowsError) FunctionName() string {
	return self.functionName
}

func (self *WindowsError) InnerError() error {
	return self.innerError
}

func (self *WindowsError) Error() string {
	return fmt.Sprintf("gowin32: %s failed: %v", self.functionName, self.innerError)
}
