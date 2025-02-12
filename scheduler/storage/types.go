/*
 *     Copyright 2022 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package storage

import (
	"time"

	"d7y.io/dragonfly/v2/scheduler/resource"
)

// Task contains content for task.
type Task struct {
	// ID is task id.
	ID string `csv:"id"`

	// URL is task download url.
	URL string `csv:"url"`

	// Type is task type.
	Type string `csv:"type"`

	// ContentLength is task total content length.
	ContentLength int64 `csv:"contentLength"`

	// TotalPieceCount is total piece count.
	TotalPieceCount int32 `csv:"totalPieceCount"`

	// BackToSourceLimit is back-to-source limit.
	BackToSourceLimit int32 `csv:"backToSourceLimit"`

	// BackToSourcePeerCount is back-to-source peer count.
	BackToSourcePeerCount int32 `csv:"backToSourcePeerCount"`

	// State is the download state of the task.
	State string `csv:"state"`

	// CreatedAt is peer create nanosecond time.
	CreatedAt int64 `csv:"createdAt"`

	// UpdatedAt is peer update nanosecond time.
	UpdatedAt int64 `csv:"updatedAt"`
}

// Host contains content for host.
type Host struct {
	// ID is host id.
	ID string `csv:"id"`

	// Type is host type.
	Type string `csv:"type"`

	// Hostname is host name.
	Hostname string `csv:"hostname"`

	// IP is host ip.
	IP string `csv:"ip"`

	// Port is grpc service port.
	Port int32 `csv:"port"`

	// DownloadPort is piece downloading port.
	DownloadPort int32 `csv:"downloadPort"`

	// Host OS.
	OS string `csv:"os"`

	// Host platform.
	Platform string `csv:"platform"`

	// Host platform family.
	PlatformFamily string `csv:"platformFamily"`

	// Host platform version.
	PlatformVersion string `csv:"platformVersion"`

	// Host kernel version.
	KernelVersion string `csv:"kernelVersion"`

	// ConcurrentUploadLimit is concurrent upload limit count.
	ConcurrentUploadLimit int32 `csv:"concurrentUploadLimit"`

	// ConcurrentUploadCount is concurrent upload count.
	ConcurrentUploadCount int32 `csv:"concurrentUploadCount"`

	// UploadCount is total upload count.
	UploadCount int64 `csv:"uploadCount"`

	// UploadFailedCount is upload failed count.
	UploadFailedCount int64 `csv:"uploadFailedCount"`

	// CPU Stat.
	CPU resource.CPU `csv:"cpu"`

	// Memory Stat.
	Memory resource.Memory `csv:"memory"`

	// Network Stat.
	Network resource.Network `csv:"network"`

	// Disk Stat.
	Disk resource.Disk `csv:"disk"`

	// Build information.
	Build resource.Build `csv:"build"`

	// CreatedAt is peer create nanosecond time.
	CreatedAt int64 `csv:"createdAt"`

	// UpdatedAt is peer update nanosecond time.
	UpdatedAt int64 `csv:"updatedAt"`
}

// Parent contains content for parent.
type Parent struct {
	// ID is peer id.
	ID string `csv:"id"`

	// Tag is peer tag.
	Tag string `csv:"tag"`

	// Application is peer application.
	Application string `csv:"application"`

	// State is the download state of the peer.
	State string `csv:"state"`

	// Cost is the task download duration of nanosecond.
	Cost int64 `csv:"cost"`

	// UploadPieceCount is upload piece count.
	UploadPieceCount int32 `csv:"uploadPieceCount"`

	// Host is peer host.
	Host Host `csv:"host"`

	// CreatedAt is peer create nanosecond time.
	CreatedAt int64 `csv:"createdAt"`

	// UpdatedAt is peer update nanosecond time.
	UpdatedAt int64 `csv:"updatedAt"`
}

// Error contains content for error.
type Error struct {
	time.Duration
	// Code is the code of error.
	Code string `csv:"code"`

	// Message is the message of error.
	Message string `csv:"message"`
}

// Download contains content for download.
type Download struct {
	// ID is peer id.
	ID string `csv:"id"`

	// Tag is peer tag.
	Tag string `csv:"tag"`

	// Application is peer application.
	Application string `csv:"application"`

	// State is the download state of the peer.
	State string `csv:"state"`

	// Error is the details of error.
	Error Error `csv:"error"`

	// Cost is the task download duration of nanosecond.
	Cost int64 `csv:"cost"`

	// Task is peer task.
	Task Task `csv:"task"`

	// Host is peer host.
	Host Host `csv:"host"`

	// Parents is peer parents.
	Parents []Parent `csv:"parents" csv[]:"20"`

	// CreatedAt is peer create nanosecond time.
	CreatedAt int64 `csv:"createdAt"`

	// UpdatedAt is peer update nanosecond time.
	UpdatedAt int64 `csv:"updatedAt"`
}
