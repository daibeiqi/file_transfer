package api

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"git-file-transfer/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FileHandler 文件处理器
type FileHandler struct {
	fileManager *storage.FileManager
}

// NewFileHandler 创建新的文件处理器
func NewFileHandler(fileManager *storage.FileManager) *FileHandler {
	return &FileHandler{
		fileManager: fileManager,
	}
}

// UploadFile 单文件上传
// @Summary 上传单个文件
// @Description 上传单个文件，支持最大100MB
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "文件"
// @Success 200 {object} UploadResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/files/upload [post]
func (h *FileHandler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("Failed to get file from form: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "Failed to get file from form",
			Error:   err.Error(),
		})
		return
	}

	// 验证文件大小
	if err := h.fileManager.ValidateFileSize(file.Size); err != nil {
		log.Printf("File size validation failed: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "File size validation failed",
			Error:   err.Error(),
		})
		return
	}

	// 验证文件类型
	if err := h.fileManager.ValidateFileType(file.Filename); err != nil {
		log.Printf("File type validation failed: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "File type validation failed",
			Error:   err.Error(),
		})
		return
	}

	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		log.Printf("Failed to open uploaded file: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to open uploaded file",
			Error:   err.Error(),
		})
		return
	}
	defer src.Close()

	// 计算文件MD5
	fileMD5, err := h.fileManager.CalculateMD5(src)
	if err != nil {
		log.Printf("Failed to calculate file MD5: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to calculate file MD5",
			Error:   err.Error(),
		})
		return
	}

	// 重置文件指针
	if _, err := src.Seek(0, 0); err != nil {
		log.Printf("Failed to seek file: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to seek file",
			Error:   err.Error(),
		})
		return
	}

	// 保存文件
	storedName, err := h.fileManager.SaveFile(src, file.Filename)
	if err != nil {
		log.Printf("Failed to save file: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to save file",
			Error:   err.Error(),
		})
		return
	}

	now := time.Now()
	fileInfo := FileInfo{
		ID:           uuid.New().String(),
		OriginalName: file.Filename,
		StoredName:   storedName,
		Size:         file.Size,
		MimeType:     file.Header.Get("Content-Type"),
		Path:         h.fileManager.GetFilePath(storedName),
		MD5:          fileMD5,
		UploadType:   UploadTypeSingle,
		Status:       StatusCompleted,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	log.Printf("File uploaded successfully: %s (ID: %s, Size: %d bytes)", fileInfo.OriginalName, fileInfo.ID, fileInfo.Size)

	c.JSON(http.StatusOK, UploadResponse{
		Success: true,
		Message: "File uploaded successfully",
		Data:    &fileInfo,
	})
}

// InitMultipartUpload 初始化分片上传
// @Summary 初始化分片上传
// @Description 初始化分片上传，返回uploadId
// @Tags files
// @Accept json
// @Produce json
// @Param request body MultipartUploadInitRequest true "上传信息"
// @Success 200 {object} MultipartUploadInitResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/files/upload-multipart/init [post]
func (h *FileHandler) InitMultipartUpload(c *gin.Context) {
	var req MultipartUploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind request: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// 验证文件大小
	if err := h.fileManager.ValidateFileSize(req.FileSize); err != nil {
		log.Printf("File size validation failed: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "File size validation failed",
			Error:   err.Error(),
		})
		return
	}

	// 验证文件类型
	if err := h.fileManager.ValidateFileType(req.FileName); err != nil {
		log.Printf("File type validation failed: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "File type validation failed",
			Error:   err.Error(),
		})
		return
	}

	// 初始化分片上传
	uploadID, chunkSize, totalChunks, err := h.fileManager.InitMultipartUpload(req.FileName, req.FileSize)
	if err != nil {
		log.Printf("Failed to initialize multipart upload: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to initialize multipart upload",
			Error:   err.Error(),
		})
		return
	}

	log.Printf("Multipart upload initialized: %s (UploadID: %s, TotalChunks: %d)", req.FileName, uploadID, totalChunks)

	c.JSON(http.StatusOK, MultipartUploadInitResponse{
		Success:     true,
		Message:     "Multipart upload initialized",
		UploadID:    uploadID,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
	})
}

// UploadChunk 上传分片
// @Summary 上传分片
// @Description 上传文件的一个分片
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param uploadId formData string true "上传ID"
// @Param chunkIndex formData int true "分片索引"
// @Param chunk formData file true "分片数据"
// @Success 200 {object} ChunkUploadResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/files/upload-multipart/chunk [post]
func (h *FileHandler) UploadChunk(c *gin.Context) {
	uploadID := c.PostForm("uploadId")
	if uploadID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "uploadId is required",
		})
		return
	}

	chunkIndexStr := c.PostForm("chunkIndex")
	if chunkIndexStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "chunkIndex is required",
		})
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "Invalid chunkIndex",
			Error:   err.Error(),
		})
		return
	}

	// 获取分片数据
	chunkFile, err := c.FormFile("chunk")
	if err != nil {
		log.Printf("Failed to get chunk from form: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "Failed to get chunk from form",
			Error:   err.Error(),
		})
		return
	}

	// 打开分片文件
	chunkData, err := chunkFile.Open()
	if err != nil {
		log.Printf("Failed to open chunk file: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to open chunk file",
			Error:   err.Error(),
		})
		return
	}
	defer chunkData.Close()

	// 读取分片数据
	chunkBytes := make([]byte, chunkFile.Size)
	if _, err := chunkData.Read(chunkBytes); err != nil {
		log.Printf("Failed to read chunk data: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to read chunk data",
			Error:   err.Error(),
		})
		return
	}

	// 保存分片
	if err := h.fileManager.SaveChunk(uploadID, chunkIndex, chunkBytes); err != nil {
		log.Printf("Failed to save chunk: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to save chunk",
			Error:   err.Error(),
		})
		return
	}

	// 获取已上传的分片列表
	uploadedChunks, _ := h.fileManager.GetUploadedChunks(uploadID)

	log.Printf("Chunk uploaded: UploadID=%s, ChunkIndex=%d, TotalUploaded=%d", uploadID, chunkIndex, len(uploadedChunks))

	c.JSON(http.StatusOK, ChunkUploadResponse{
		Success:        true,
		Message:        "Chunk uploaded successfully",
		ChunkIndex:     chunkIndex,
		UploadedChunks: uploadedChunks,
	})
}

// CompleteMultipartUpload 完成分片上传
// @Summary 完成分片上传
// @Description 合并所有分片并完成上传
// @Tags files
// @Accept json
// @Produce json
// @Param request body MultipartCompleteRequest true "完成请求"
// @Success 200 {object} MultipartCompleteResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/files/upload-multipart/complete [post]
func (h *FileHandler) CompleteMultipartUpload(c *gin.Context) {
	var req MultipartCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind request: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// 获取上传状态以获取文件名
	status, _ := h.fileManager.GetUploadStatus(req.UploadID)
	if status == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Message: "Upload not found",
		})
		return
	}

	// 从元数据文件中读取原始文件名
	originalName := h.fileManager.GetUploadFileName(req.UploadID)

	// 合并分片
	storedName, err := h.fileManager.MergeChunks(req.UploadID, originalName)
	if err != nil {
		log.Printf("Failed to merge chunks: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to merge chunks",
			Error:   err.Error(),
		})
		return
	}

	now := time.Now()
	fileInfo := FileInfo{
		ID:           uuid.New().String(),
		OriginalName: originalName,
		StoredName:   storedName,
		MimeType:     "application/octet-stream",
		Path:         h.fileManager.GetFilePath(storedName),
		MD5:          req.MD5,
		UploadType:   UploadTypeMultipart,
		Status:       StatusCompleted,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	log.Printf("Multipart upload completed: %s (ID: %s)", fileInfo.OriginalName, fileInfo.ID)

	c.JSON(http.StatusOK, MultipartCompleteResponse{
		Success: true,
		Message: "Multipart upload completed",
		Data:    &fileInfo,
	})
}

// GetUploadStatus 获取上传状态
// @Summary 获取上传状态
// @Description 获取分片上传的当前状态
// @Tags files
// @Produce json
// @Param uploadId path string true "上传ID"
// @Success 200 {object} UploadStatusResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/files/upload-multipart/status/{uploadId} [get]
func (h *FileHandler) GetUploadStatus(c *gin.Context) {
	uploadID := c.Param("uploadId")
	if uploadID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "uploadId is required",
		})
		return
	}

	status, err := h.fileManager.GetUploadStatus(uploadID)
	if err != nil {
		log.Printf("Failed to get upload status: %v", err)
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Message: "Upload not found",
			Error:   err.Error(),
		})
		return
	}

	// 转换storage.UploadStatusResponse到api.UploadStatusResponse
	apiStatus := UploadStatusResponse{
		Success:        true,
		UploadedSize:   status.UploadedSize,
		TotalSize:      status.TotalSize,
		UploadedChunks: status.UploadedChunks,
		TotalChunks:    status.TotalChunks,
		Percentage:     status.Percentage,
		Status:         string(status.Status),
	}

	c.JSON(http.StatusOK, apiStatus)
}

// DownloadFile 下载文件
// @Summary 下载文件
// @Description 下载已上传的文件
// @Tags files
// @Produce application/octet-stream
// @Param id path string true "文件ID"
// @Success 200 {file} binary
// @Failure 404 {object} ErrorResponse
// @Router /api/files/{id}/download [get]
func (h *FileHandler) DownloadFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "File ID is required",
		})
		return
	}

	// 尝试直接查找文件
	filePath := filepath.Join(h.fileManager.GetBaseDir(), fileID)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 尝试查找完整的文件名
		files, err := os.ReadDir(h.fileManager.GetBaseDir())
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Success: false,
				Message: "Failed to read files",
				Error:   err.Error(),
			})
			return
		}

		var foundFile string
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			// 检查是否包含ID
			if filepath.Base(f.Name()) == fileID || filepath.Ext(f.Name()) == "" && f.Name() == fileID {
				foundFile = f.Name()
				break
			}
			// 检查文件名是否包含ID作为UUID部分
			if strings.Contains(f.Name(), fileID) {
				foundFile = f.Name()
				break
			}
		}

		if foundFile == "" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Message: "File not found",
			})
			return
		}

		filePath = filepath.Join(h.fileManager.GetBaseDir(), foundFile)
	}

	c.File(filePath)
	log.Printf("File downloaded: %s", filepath.Base(filePath))
}

// ListFiles 列出所有文件
// @Summary 列出所有文件
// @Description 获取已上传的文件列表
// @Tags files
// @Produce json
// @Success 200 {object} FileListResponse
// @Router /api/files [get]
func (h *FileHandler) ListFiles(c *gin.Context) {
	files, err := h.fileManager.ListFiles()
	if err != nil {
		log.Printf("Failed to list files: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to list files",
			Error:   err.Error(),
		})
		return
	}

	// 转换storage.FileInfo到api.FileInfo
	data := make([]interface{}, len(files))
	for i, f := range files {
		data[i] = FileInfo{
			ID:           f.ID,
			OriginalName: f.OriginalName,
			StoredName:   f.StoredName,
			Size:         f.Size,
			MimeType:     f.MimeType,
			Path:         f.Path,
			MD5:          f.MD5,
			UploadType:   UploadTypeSingle,
			Status:       StatusCompleted,
			CreatedAt:    f.CreatedAt,
			UpdatedAt:    f.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, FileListResponse{
		Success: true,
		Message: "Files listed successfully",
		Data:    data,
		Total:   int64(len(files)),
	})
}

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine, fileManager *storage.FileManager) {
	fileHandler := NewFileHandler(fileManager)

	// API路由组
	api := r.Group("/api")
	{
		// 文件上传路由
		files := api.Group("/files")
		{
			files.GET("", fileHandler.ListFiles)
			files.POST("/upload", fileHandler.UploadFile)
			files.GET("/:id/download", fileHandler.DownloadFile)

			// 分片上传路由
			multipart := files.Group("/upload-multipart")
			{
				multipart.POST("/init", fileHandler.InitMultipartUpload)
				multipart.POST("/chunk", fileHandler.UploadChunk)
				multipart.POST("/complete", fileHandler.CompleteMultipartUpload)
				multipart.GET("/status/:uploadId", fileHandler.GetUploadStatus)
			}
		}
	}

	// 静态文件服务（上传的文件）
	r.Static("/uploads", fileManager.GetBaseDir())
}
