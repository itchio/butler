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

type ProcessorArchitecture uint16

const (
	ProcessorArchitectureIntel   ProcessorArchitecture = wrappers.PROCESSOR_ARCHITECTURE_INTEL
	ProcessorArchitectureMIPS    ProcessorArchitecture = wrappers.PROCESSOR_ARCHITECTURE_MIPS
	ProcessorArchitectureAlpha   ProcessorArchitecture = wrappers.PROCESSOR_ARCHITECTURE_ALPHA
	ProcessorArchitecturePowerPC ProcessorArchitecture = wrappers.PROCESSOR_ARCHITECTURE_PPC
	ProcessorArchitectureARM     ProcessorArchitecture = wrappers.PROCESSOR_ARCHITECTURE_ARM
	ProcessorArchitectureIA64    ProcessorArchitecture = wrappers.PROCESSOR_ARCHITECTURE_IA64
	ProcessorArchitectureAMD64   ProcessorArchitecture = wrappers.PROCESSOR_ARCHITECTURE_AMD64
)

type ProcessorInfo struct {
	ProcessorArchitecture ProcessorArchitecture
	NumberOfProcessors    uint
	ProcessorLevel        uint
	ProcessorRevision     uint
}

func GetProcessorInfo() *ProcessorInfo {
	var si wrappers.SYSTEM_INFO
	wrappers.GetSystemInfo(&si)
	return &ProcessorInfo{
		ProcessorArchitecture: ProcessorArchitecture(si.ProcessorArchitecture),
		NumberOfProcessors:    uint(si.NumberOfProcessors),
		ProcessorLevel:        uint(si.ProcessorLevel),
		ProcessorRevision:     uint(si.ProcessorRevision),
	}
}
