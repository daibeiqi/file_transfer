package api

import (
	"time"
)

// UploadStatus 上传状态
type UploadStatus string

const (
	StatusPending   UploadStatus = "pending"
	StatusUploading UploadStatus = "uploading"
	StatusCompleted UploadStatus = "completed"
	StatusFailed    UploadStatus = "failed"
	StatusCanceled  UploadStatus = "canceled"
)

// UploadType 上传类型
type UploadType string

const (
	UploadTypeSingle    UploadType = "single"
	UploadTypeMultipart UploadType = "multipart"
)

// FileInfo 文件信息
type FileInfo struct {
	ID           string      `json:"id"`
	OriginalName string      `json:"originalName"`
	StoredName   string      `json:"storedName"`
	Size         int64       `json:"size"`
	MimeType     string      `json:"mimeType"`
	Path         string      `json:"path,omitempty"`
	MD5          string      `json:"md5,omitempty"`
	UploadType   UploadType  `json:"uploadType"`
	Status       UploadStatus `json:"status"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}

// UploadRequest 单文件上传请求
type UploadRequest struct {
	File        []byte `form:"file" binding:"required"`
	FileName    string `form:"fileName" binding:"required"`
	ContentType string `form:"contentType"`
}

// UploadResponse 单文件上传响应
type UploadResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message"`
	Data    *FileInfo `json:"data,omitempty"`
}

// MultipartUploadInitRequest 分片上传初始化请求
type MultipartUploadInitRequest struct {
	FileName    string `json:"fileName" binding:"required"`
	FileSize    int64  `json:"fileSize" binding:"required"`
	ChunkSize   int64  `json:"chunkSize"`
	ContentType string `json:"contentType"`
	MD5         string `json:"md5"`
}

// MultipartUploadInitResponse 分片上传初始化响应
type MultipartUploadInitResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	UploadID    string `json:"uploadId,omitempty"`
	ChunkSize   int64  `json:"chunkSize,omitempty"`
	TotalChunks int    `json:"totalChunks,omitempty"`
}

// ChunkUploadRequest 分片上传请求
type ChunkUploadRequest struct {
	UploadID   string `form:"uploadId" binding:"required"`
	ChunkIndex int    `form:"chunkIndex" binding:"required"`
	ChunkMD5   string `form:"chunkMd5"`
}

// ChunkUploadResponse 分片上传响应
type ChunkUploadResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	ChunkIndex     int    `json:"chunkIndex,omitempty"`
	UploadedChunks []int  `json:"uploadedChunks,omitempty"`
}

// MultipartCompleteRequest 分片上传完成请求
type MultipartCompleteRequest struct {
	UploadID string `json:"uploadId" binding:"required"`
	MD5      string `json:"md5"`
}

// MultipartCompleteResponse 分片上传完成响应
type MultipartCompleteResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message"`
	Data    *FileInfo `json:"data,omitempty"`
}

// UploadStatusResponse 上传状态响应
type UploadStatusResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	Status         string `json:"status"`
	UploadedSize   int64  `json:"uploadedSize"`
	TotalSize      int64  `json:"totalSize"`
	UploadedChunks []int  `json:"uploadedChunks,omitempty"`
	TotalChunks    int    `json:"totalChunks,omitempty"`
	Percentage     float64 `json:"percentage"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// FileListResponse 文件列表响应
type FileListResponse struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Data    []interface{} `json:"data,omitempty"`
	Total   int64         `json:"total"`
}
