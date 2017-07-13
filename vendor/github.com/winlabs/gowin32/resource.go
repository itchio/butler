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

type ResourceType uintptr

func CustomResourceType(resourceTypeName string) ResourceType {
	return ResourceType(unsafe.Pointer(syscall.StringToUTF16Ptr(resourceTypeName)))
}

var (
	ResourceTypeCursor        ResourceType = ResourceType(wrappers.RT_CURSOR)
	ResourceTypeBitmap        ResourceType = ResourceType(wrappers.RT_BITMAP)
	ResourceTypeIcon          ResourceType = ResourceType(wrappers.RT_ICON)
	ResourceTypeMenu          ResourceType = ResourceType(wrappers.RT_MENU)
	ResourceTypeDialog        ResourceType = ResourceType(wrappers.RT_DIALOG)
	ResourceTypeString        ResourceType = ResourceType(wrappers.RT_STRING)
	ResourceTypeFontDir       ResourceType = ResourceType(wrappers.RT_FONTDIR)
	ResourceTypeFont          ResourceType = ResourceType(wrappers.RT_FONT)
	ResourceTypeAccelerator   ResourceType = ResourceType(wrappers.RT_ACCELERATOR)
	ResourceTypeRCData        ResourceType = ResourceType(wrappers.RT_RCDATA)
	ResourceTypeMessageTable  ResourceType = ResourceType(wrappers.RT_MESSAGETABLE)
	ResourceTypeGroupCursor   ResourceType = ResourceType(wrappers.RT_GROUP_CURSOR)
	ResourceTypeGroupIcon     ResourceType = ResourceType(wrappers.RT_GROUP_ICON)
	ResourceTypeVersion       ResourceType = ResourceType(wrappers.RT_VERSION)
	ResourceTypeDialogInclude ResourceType = ResourceType(wrappers.RT_DLGINCLUDE)
	ResourceTypePlugPlay      ResourceType = ResourceType(wrappers.RT_PLUGPLAY)
	ResourceTypeVxD           ResourceType = ResourceType(wrappers.RT_VXD)
	ResourceTypeAniCursor     ResourceType = ResourceType(wrappers.RT_ANICURSOR)
	ResourceTypeAniIcon       ResourceType = ResourceType(wrappers.RT_ANIICON)
	ResourceTypeHTML          ResourceType = ResourceType(wrappers.RT_HTML)
	ResourceTypeManifest      ResourceType = ResourceType(wrappers.RT_MANIFEST)
)

type ResourceId uintptr

func IntResourceId(resourceId uint) ResourceId {
	return ResourceId(wrappers.MAKEINTRESOURCE(uint16(resourceId)))
}

func StringResourceId(resourceId string) ResourceId {
	return ResourceId(unsafe.Pointer(syscall.StringToUTF16Ptr(resourceId)))
}

type ResourceUpdate struct {
	handle syscall.Handle
}

func NewResourceUpdate(fileName string, deleteExistingResources bool) (*ResourceUpdate, error) {
	hUpdate, err := wrappers.BeginUpdateResource(syscall.StringToUTF16Ptr(fileName), deleteExistingResources)
	if err != nil {
		return nil, NewWindowsError("BeginUpdateResource", err)
	}
	return &ResourceUpdate{handle: hUpdate}, nil
}

func (self *ResourceUpdate) Close() error {
	if self.handle != 0 {
		if err := wrappers.EndUpdateResource(self.handle, true); err != nil {
			return NewWindowsError("EndUpdateResource", err)
		}
		self.handle = 0
	}
	return nil
}

func (self *ResourceUpdate) Save() error {
	if err := wrappers.EndUpdateResource(self.handle, false); err != nil {
		return NewWindowsError("EndUpdateResource", err)
	}
	self.handle = 0
	return nil
}

func (self *ResourceUpdate) Update(resourceType ResourceType, resourceId ResourceId, language Language, data []byte) error {
	err := wrappers.UpdateResource(
		self.handle,
		uintptr(resourceType),
		uintptr(resourceId),
		uint16(language),
		&data[0],
		uint32(len(data)))
	if err != nil {
		return NewWindowsError("UpdateResource", err)
	}
	return nil
}

func (self *ResourceUpdate) Delete(resourceType ResourceType, resourceId ResourceId, language Language) error {
	err := wrappers.UpdateResource(
		self.handle,
		uintptr(resourceType),
		uintptr(resourceId),
		uint16(language),
		nil,
		0)
	if err != nil {
		return NewWindowsError("UpdateResource", err)
	}
	return nil
}
