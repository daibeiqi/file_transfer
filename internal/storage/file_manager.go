package storage

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// MaxFileSize 最大文件大小 100MB
	MaxFileSize = 100 * 1024 * 1024
	// DefaultChunkSize 默认分片大小 5MB
	DefaultChunkSize = 5 * 1024 * 1024
)

// AllowedExtensions 允许的文件扩展名
var AllowedExtensions = []string{
	".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", // 图片
	".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", // 文档
	".zip", ".rar", ".7z", ".tar", ".gz", // 压缩包
}

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

// UploadStatusResponse 上传状态响应
type UploadStatusResponse struct {
	Success        bool         `json:"success"`
	UploadedSize   int64        `json:"uploadedSize"`
	TotalSize      int64        `json:"totalSize"`
	UploadedChunks []int        `json:"uploadedChunks,omitempty"`
	TotalChunks    int          `json:"totalChunks,omitempty"`
	Percentage     float64      `json:"percentage"`
	Status         UploadStatus `json:"status"`
}

// FileInfo 文件信息
type FileInfo struct {
	ID           string       `json:"id"`
	OriginalName string       `json:"originalName"`
	StoredName   string       `json:"storedName"`
	Size         int64        `json:"size"`
	MimeType     string       `json:"mimeType"`
	Path         string       `json:"path,omitempty"`
	MD5          string       `json:"md5,omitempty"`
	Status       UploadStatus `json:"status"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

// FileManager 文件管理器
type FileManager struct {
	baseDir      string
	chunkDir     string
	tempDir      string
	uploadLocks  sync.Map
	uploadStatus sync.Map
}

// NewFileManager 创建新的文件管理器
func NewFileManager(baseDir string) *FileManager {
	chunkDir := filepath.Join(baseDir, "chunks")
	tempDir := filepath.Join(baseDir, "temp")

	// 确保目录存在
	dirs := []string{baseDir, chunkDir, tempDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	return &FileManager{
		baseDir:  baseDir,
		chunkDir: chunkDir,
		tempDir:  tempDir,
	}
}

// GenerateUniqueFilename 生成唯一文件名
func (fm *FileManager) GenerateUniqueFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	base := strings.TrimSuffix(originalName, ext)
	timestamp := time.Now().Format("20060102_150405")
	uuidStr := uuid.New().String()[:8]
	return fmt.Sprintf("%s_%s_%s%s", base, timestamp, uuidStr, ext)
}

// ValidateFileSize 验证文件大小
func (fm *FileManager) ValidateFileSize(size int64) error {
	if size > MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum limit of %d bytes", size, MaxFileSize)
	}
	if size == 0 {
		return fmt.Errorf("file size is zero")
	}
	return nil
}

// ValidateFileType 验证文件类型
func (fm *FileManager) ValidateFileType(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowedExt := range AllowedExtensions {
		if ext == allowedExt {
			return nil
		}
	}
	return fmt.Errorf("file type %s is not allowed", ext)
}

// CalculateMD5 计算文件MD5
func (fm *FileManager) CalculateMD5(file io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SaveFile 保存文件
func (fm *FileManager) SaveFile(file multipart.File, filename string) (string, error) {
	storedName := fm.GenerateUniqueFilename(filename)
	filePath := filepath.Join(fm.baseDir, storedName)

	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		// 删除已创建的文件
		os.Remove(filePath)
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return storedName, nil
}

// GetFilePath 获取文件路径
func (fm *FileManager) GetFilePath(storedName string) string {
	return filepath.Join(fm.baseDir, storedName)
}

// GetBaseDir 获取基础目录
func (fm *FileManager) GetBaseDir() string {
	return fm.baseDir
}

// GetChunkDir 获取分片目录
func (fm *FileManager) GetChunkDir() string {
	return fm.chunkDir
}

// DeleteFile 删除文件
func (fm *FileManager) DeleteFile(storedName string) error {
	filePath := fm.GetFilePath(storedName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// InitMultipartUpload 初始化分片上传
func (fm *FileManager) InitMultipartUpload(filename string, fileSize int64) (uploadID string, chunkSize int64, totalChunks int, err error) {
	uploadID = uuid.New().String()
	chunkSize = DefaultChunkSize

	// 计算总片数
	totalChunks = int(fileSize / chunkSize)
	if fileSize%chunkSize > 0 {
		totalChunks++
	}

	// 创建上传目录
	uploadDir := filepath.Join(fm.chunkDir, uploadID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", 0, 0, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// 初始化上传状态
	status := &UploadProgress{
		UploadID:       uploadID,
		FileName:       filename,
		TotalSize:      fileSize,
		UploadedSize:   0,
		TotalChunks:    totalChunks,
		UploadedChunks: []int{},
		Status:         StatusPending,
		StartTime:      time.Now(),
		LastUpdateTime: time.Now(),
	}
	fm.uploadStatus.Store(uploadID, status)

	// 保存元数据文件以便后续合并时获取原始文件名
	metadata := map[string]interface{}{
		"fileName":    filename,
		"fileSize":    fileSize,
		"chunkSize":   chunkSize,
		"totalChunks": totalChunks,
		"uploadID":    uploadID,
		"createdAt":   time.Now().Format(time.RFC3339),
	}
	metadataPath := filepath.Join(uploadDir, ".metadata")
	metadataBytes, _ := json.Marshal(metadata)
	_ = os.WriteFile(metadataPath, metadataBytes, 0644)

	return uploadID, chunkSize, totalChunks, nil
}

// SaveChunk 保存分片
func (fm *FileManager) SaveChunk(uploadID string, chunkIndex int, chunkData []byte) error {
	// 获取锁以确保并发安全
	mu, _ := fm.uploadLocks.LoadOrStore(uploadID, &sync.Mutex{})
	lock := mu.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	// 保存分片文件
	chunkFilename := fmt.Sprintf("chunk_%d", chunkIndex)
	chunkPath := filepath.Join(fm.chunkDir, uploadID, chunkFilename)

	if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
		return fmt.Errorf("failed to save chunk %d: %w", chunkIndex, err)
	}

	// 更新上传状态
	if status, ok := fm.uploadStatus.Load(uploadID); ok {
		progress := status.(*UploadProgress)
		progress.UploadedSize += int64(len(chunkData))
		progress.UploadedChunks = append(progress.UploadedChunks, chunkIndex)
		progress.LastUpdateTime = time.Now()
		progress.Status = StatusUploading

		if progress.UploadedSize >= progress.TotalSize {
			progress.Status = StatusCompleted
		}

		// 计算百分比
		progress.Percentage = float64(progress.UploadedSize) / float64(progress.TotalSize) * 100
		fm.uploadStatus.Store(uploadID, progress)
	}

	return nil
}

// GetUploadedChunks 获取已上传的分片
func (fm *FileManager) GetUploadedChunks(uploadID string) ([]int, error) {
	uploadDir := filepath.Join(fm.chunkDir, uploadID)
	var uploadedChunks []int

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}, nil
		}
		return nil, fmt.Errorf("failed to read upload directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			var index int
			_, err := fmt.Sscanf(entry.Name(), "chunk_%d", &index)
			if err == nil {
				uploadedChunks = append(uploadedChunks, index)
			}
		}
	}

	return uploadedChunks, nil
}

// GetUploadFileName 获取上传的原始文件名
func (fm *FileManager) GetUploadFileName(uploadID string) string {
	uploadDir := filepath.Join(fm.chunkDir, uploadID)
	metadataPath := filepath.Join(uploadDir, ".metadata")

	if data, err := os.ReadFile(metadataPath); err == nil {
		var metadata struct {
			FileName string `json:"fileName"`
		}
		if err := json.Unmarshal(data, &metadata); err == nil && metadata.FileName != "" {
			return metadata.FileName
		}
	}

	// 尝试从内存中获取
	if status, ok := fm.uploadStatus.Load(uploadID); ok {
		progress := status.(*UploadProgress)
		return progress.FileName
	}

	return "unknown_file.bin"
}

// MergeChunks 合并分片
func (fm *FileManager) MergeChunks(uploadID string, filename string) (string, error) {
	// 获取锁以确保并发安全
	mu, _ := fm.uploadLocks.LoadOrStore(uploadID, &sync.Mutex{})
	lock := mu.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	uploadDir := filepath.Join(fm.chunkDir, uploadID)
	storedName := fm.GenerateUniqueFilename(filename)
	outputPath := filepath.Join(fm.baseDir, storedName)

	// 打开输出文件
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// 读取并合并所有分片
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		return "", fmt.Errorf("failed to read upload directory: %w", err)
	}

	// 按分片索引排序
	chunkFiles := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "chunk_") {
			chunkFiles = append(chunkFiles, entry)
		}
	}

	// 对分片文件进行排序
	for i := 0; i < len(chunkFiles); i++ {
		for j := i + 1; j < len(chunkFiles); j++ {
			var idx1, idx2 int
			fmt.Sscanf(chunkFiles[i].Name(), "chunk_%d", &idx1)
			fmt.Sscanf(chunkFiles[j].Name(), "chunk_%d", &idx2)
			if idx1 > idx2 {
				chunkFiles[i], chunkFiles[j] = chunkFiles[j], chunkFiles[i]
			}
		}
	}

	// 按顺序合并分片
	for _, entry := range chunkFiles {
		chunkPath := filepath.Join(uploadDir, entry.Name())
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			os.Remove(outputPath)
			return "", fmt.Errorf("failed to read chunk %s: %w", entry.Name(), err)
		}

		if _, err := outputFile.Write(chunkData); err != nil {
			os.Remove(outputPath)
			return "", fmt.Errorf("failed to write chunk %s: %w", entry.Name(), err)
		}
	}

	// 清理分片目录
	_ = os.RemoveAll(uploadDir)

	// 清除上传状态
	fm.uploadStatus.Delete(uploadID)
	fm.uploadLocks.Delete(uploadID)

	return storedName, nil
}

// GetUploadStatus 获取上传状态
func (fm *FileManager) GetUploadStatus(uploadID string) (*UploadStatusResponse, error) {
	status, ok := fm.uploadStatus.Load(uploadID)
	if !ok {
		// 尝试从已上传的分片计算状态
		uploadedChunks, err := fm.GetUploadedChunks(uploadID)
		if err != nil {
			return nil, fmt.Errorf("upload not found: %s", uploadID)
		}

		uploadDir := filepath.Join(fm.chunkDir, uploadID)
		entries, _ := os.ReadDir(uploadDir)
		totalChunks := len(entries)

		// 计算已上传大小
		var uploadedSize int64
		for _, entry := range entries {
			if !entry.IsDir() {
				info, _ := entry.Info()
				uploadedSize += info.Size()
			}
		}

		return &UploadStatusResponse{
			Success:        true,
			Status:         StatusUploading,
			UploadedChunks: uploadedChunks,
			TotalChunks:    totalChunks,
		}, nil
	}

	progress := status.(*UploadProgress)
	uploadedChunks := make([]int, len(progress.UploadedChunks))
	copy(uploadedChunks, progress.UploadedChunks)

	return &UploadStatusResponse{
		Success:        true,
		Status:         progress.Status,
		UploadedSize:   progress.UploadedSize,
		TotalSize:      progress.TotalSize,
		UploadedChunks: uploadedChunks,
		TotalChunks:    progress.TotalChunks,
		Percentage:     progress.Percentage,
	}, nil
}

// AbortMultipartUpload 中止分片上传
func (fm *FileManager) AbortMultipartUpload(uploadID string) error {
	uploadDir := filepath.Join(fm.chunkDir, uploadID)

	if err := os.RemoveAll(uploadDir); err != nil && !os.IsNotExist(err) {
		return err
	}

	// 清除上传状态
	fm.uploadStatus.Delete(uploadID)
	fm.uploadLocks.Delete(uploadID)

	return nil
}

// ListFiles 列出所有文件
func (fm *FileManager) ListFiles() ([]FileInfo, error) {
	entries, err := os.ReadDir(fm.baseDir)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileInfo := FileInfo{
			ID:           strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			StoredName:   entry.Name(),
			OriginalName: entry.Name(),
			Size:         info.Size(),
			Path:         fm.GetFilePath(entry.Name()),
			Status:       StatusCompleted,
			CreatedAt:    info.ModTime(),
			UpdatedAt:    info.ModTime(),
		}
		files = append(files, fileInfo)
	}

	return files, nil
}
