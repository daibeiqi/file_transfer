package types

import "time"

// UploadStatus 上传状态
type UploadStatus string

const (
	StatusPending   UploadStatus = "pending"
	StatusUploading UploadStatus = "uploading"
	StatusCompleted UploadStatus = "completed"
	StatusFailed    UploadStatus = "failed"
	StatusCanceled  UploadStatus = "canceled"
)

// UploadProgress 上传进度
type UploadProgress struct {
	UploadID       string       `json:"uploadId"`
	FileName       string       `json:"fileName"`
	TotalSize      int64        `json:"totalSize"`
	UploadedSize   int64        `json:"uploadedSize"`
	TotalChunks    int          `json:"totalChunks"`
	UploadedChunks []int        `json:"uploadedChunks"`
	Status         UploadStatus `json:"status"`
	Percentage     float64      `json:"percentage"`
	Speed          int64        `json:"speed"` // bytes per second
	StartTime      time.Time    `json:"startTime"`
	LastUpdateTime time.Time    `json:"lastUpdateTime"`
}
