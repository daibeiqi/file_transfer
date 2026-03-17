import { useState, useRef, useCallback, useEffect } from 'react';
import { fileService } from '@/services/api';
import type { UploadItem, UploadProgress, UploadStatus, FileInfo } from '@/types';

// 常量
const CHUNK_SIZE = 5 * 1024 * 1024; // 5MB
const MAX_FILE_SIZE = 100 * 1024 * 1024; // 100MB
const ALLOWED_EXTENSIONS = [
  '.jpg', '.jpeg', '.png', '.gif', '.webp', '.svg', // 图片
  '.pdf', '.doc', '.docx', '.xls', '.xlsx', '.ppt', '.pptx', '.txt', // 文档
  '.zip', '.rar', '.7z', '.tar', '.gz', // 压缩包
];

// 验证文件类型
const isValidFileType = (fileName: string): boolean => {
  const ext = '.' + fileName.split('.').pop()?.toLowerCase();
  return ALLOWED_EXTENSIONS.includes(ext);
};

// 格式化文件大小
const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
};

interface FileUploaderProps {
  onUploadComplete?: (fileInfo: FileInfo) => void;
  maxFiles?: number;
  maxSize?: number;
}

export function FileUploader({ onUploadComplete, maxFiles = 10, maxSize = MAX_FILE_SIZE }: FileUploaderProps) {
  const [uploadItems, setUploadItems] = useState<UploadItem[]>([]);
  const [isDragging, setIsDragging] = useState(false);
  const [uploadedFiles, setUploadedFiles] = useState<FileInfo[]>([]);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // 加载已上传的文件列表
  useEffect(() => {
    loadFiles();
  }, []);

  const loadFiles = async () => {
    try {
      const response = await fileService.listFiles();
      if (response.success && response.data) {
        setUploadedFiles(response.data);
      }
    } catch (error) {
      console.error('Failed to load files:', error);
    }
  };

  // 处理文件选择
  const handleFileSelect = (files: FileList | null) => {
    if (!files) return;

    const fileArray = Array.from(files);
    const validFiles: File[] = [];

    // 验证文件
    for (const file of fileArray) {
      // 检查文件大小
      if (file.size > maxSize) {
        alert(`文件 "${file.name}" 超过最大大小限制 (${formatFileSize(maxSize)})`);
        continue;
      }

      // 检查文件类型
      if (!isValidFileType(file.name)) {
        alert(`不支持的文件类型: "${file.name}"`);
        continue;
      }

      validFiles.push(file);
    }

    // 检查最大文件数量
    if (uploadItems.length + validFiles.length > maxFiles) {
      alert(`最多只能上传 ${maxFiles} 个文件`);
      return;
    }

    // 添加到上传队列
    const newUploadItems: UploadItem[] = validFiles.map((file) => ({
      id: Math.random().toString(36).substring(7),
      file,
      progress: {
        fileId: Math.random().toString(36).substring(7),
        fileName: file.name,
        fileSize: file.size,
        uploadedSize: 0,
        totalChunks: Math.ceil(file.size / CHUNK_SIZE),
        uploadedChunks: [],
        status: 'pending',
        percentage: 0,
        speed: 0,
        startTime: Date.now(),
        lastUpdateTime: Date.now(),
      },
    }));

    setUploadItems((prev) => [...prev, ...newUploadItems]);

    // 开始上传
    newUploadItems.forEach((item) => {
      startUpload(item);
    });
  };

  // 开始上传文件
  const startUpload = async (item: UploadItem) => {
    const { file, id } = item;

    // 根据文件大小决定上传方式
    if (file.size > 10 * 1024 * 1024) {
      // 大文件使用分片上传
      await uploadMultipart(id, file);
    } else {
      // 小文件使用单文件上传
      await uploadSingle(id, file);
    }
  };

  // 单文件上传
  const uploadSingle = async (itemId: string, file: File) => {
    setUploadItems((prev) =>
      prev.map((item) =>
        item.id === itemId
          ? {
              ...item,
              progress: { ...item.progress, status: 'uploading' },
            }
          : item,
      ),
    );

    try {
      const response = await fileService.uploadFile(file, (progress) => {
        setUploadItems((prev) =>
          prev.map((item) =>
            item.id === itemId
              ? {
                  ...item,
                  progress: {
                    ...item.progress,
                    uploadedSize: (file.size * progress) / 100,
                    percentage: progress,
                    lastUpdateTime: Date.now(),
                  },
                }
              : item,
          ),
        );
      });

      if (response.success && response.data) {
        setUploadItems((prev) => prev.filter((item) => item.id !== itemId));
        setUploadedFiles((prev) => [response.data!, ...prev]);
        onUploadComplete?.(response.data);
      }
    } catch (error) {
      console.error('Upload failed:', error);
      setUploadItems((prev) =>
        prev.map((item) =>
          item.id === itemId
            ? {
                ...item,
                progress: { ...item.progress, status: 'failed' },
              }
            : item,
        ),
      );
    }
  };

  // 分片上传
  const uploadMultipart = async (itemId: string, file: File) => {
    setUploadItems((prev) =>
      prev.map((item) =>
        item.id === itemId
          ? {
              ...item,
              progress: { ...item.progress, status: 'uploading' },
            }
          : item,
      ),
    );

    try {
      // 初始化分片上传
      const initResponse = await fileService.initMultipartUpload(file.name, file.size);
      if (!initResponse.success || !initResponse.uploadId) {
        throw new Error('Failed to initialize multipart upload');
      }

      const uploadId = initResponse.uploadId;
      const totalChunks = Math.ceil(file.size / CHUNK_SIZE);

      // 上传所有分片
      for (let i = 0; i < totalChunks; i++) {
        const start = i * CHUNK_SIZE;
        const end = Math.min(start + CHUNK_SIZE, file.size);
        const chunk = file.slice(start, end);

        await fileService.uploadChunk(uploadId, i, chunk);

        // 更新进度
        const uploadedSize = Math.min((i + 1) * CHUNK_SIZE, file.size);
        setUploadItems((prev) =>
          prev.map((item) =>
            item.id === itemId
              ? {
                  ...item,
                  progress: {
                    ...item.progress,
                    uploadedSize,
                    uploadedChunks: [...item.progress.uploadedChunks, i],
                    percentage: Math.round((uploadedSize / file.size) * 100),
                    lastUpdateTime: Date.now(),
                  },
                }
              : item,
          ),
        );
      }

      // 完成上传
      const completeResponse = await fileService.completeMultipartUpload(uploadId);
      if (completeResponse.success && completeResponse.data) {
        setUploadItems((prev) => prev.filter((item) => item.id !== itemId));
        setUploadedFiles((prev) => [completeResponse.data!, ...prev]);
        onUploadComplete?.(completeResponse.data);
      }
    } catch (error) {
      console.error('Multipart upload failed:', error);
      setUploadItems((prev) =>
        prev.map((item) =>
          item.id === itemId
            ? {
                ...item,
                progress: { ...item.progress, status: 'failed' },
              }
            : item,
        ),
      );
    }
  };

  // 处理拖拽事件
  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    handleFileSelect(e.dataTransfer.files);
  };

  // 点击选择文件
  const handleClick = () => {
    fileInputRef.current?.click();
  };

  // 取消上传
  const handleCancel = (id: string) => {
    setUploadItems((prev) =>
      prev.map((item) =>
        item.id === id
          ? { ...item, progress: { ...item.progress, status: 'canceled' } }
          : item,
      ),
    );
    setTimeout(() => {
      setUploadItems((prev) => prev.filter((item) => item.id !== id));
    }, 1000);
  };

  // 删除已上传文件
  const handleDelete = (id: string) => {
    setUploadedFiles((prev) => prev.filter((file) => file.id !== id));
  };

  return (
    <div className="file-uploader">
      {/* 上传区域 */}
      <div
        className={`upload-zone ${isDragging ? 'dragging' : ''}`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={handleClick}
      >
        <svg className="upload-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
          />
        </svg>
        <p className="upload-text">拖拽文件到这里，或点击选择文件</p>
        <p className="upload-hint">
          支持 JPG、PNG、PDF、Word、Excel、PPT、TXT、ZIP 等格式，最大 100MB
        </p>
        <input
          ref={fileInputRef}
          type="file"
          multiple
          onChange={(e) => handleFileSelect(e.target.files)}
          style={{ display: 'none' }}
        />
      </div>

      {/* 上传队列 */}
      {uploadItems.length > 0 && (
        <div className="upload-queue">
          <h3>上传队列</h3>
          {uploadItems.map((item) => (
            <div key={item.id} className="upload-item">
              <div className="upload-item-info">
                <span className="file-name">{item.progress.fileName}</span>
                <span className="file-size">{formatFileSize(item.progress.fileSize)}</span>
              </div>
              <div className="progress-bar-container">
                <div
                  className="progress-bar"
                  style={{ width: `${item.progress.percentage}%` }}
                />
              </div>
              <div className="upload-item-status">
                <span>{item.progress.percentage}%</span>
                {item.progress.status === 'uploading' && (
                  <button
                    className="cancel-btn"
                    onClick={() => handleCancel(item.id)}
                  >
                    取消
                  </button>
                )}
                {item.progress.status === 'failed' && (
                  <span className="error-text">上传失败</span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 已上传文件列表 */}
      {uploadedFiles.length > 0 && (
        <div className="uploaded-files">
          <h3>已上传文件</h3>
          <div className="file-list">
            {uploadedFiles.map((file) => (
              <div key={file.id} className="file-item">
                <div className="file-item-info">
                  <span className="file-name">{file.originalName}</span>
                  <span className="file-size">{formatFileSize(file.size)}</span>
                </div>
                <div className="file-item-actions">
                  <button
                    className="download-btn"
                    onClick={() => fileService.downloadFile(file.id, file.originalName)}
                  >
                    下载
                  </button>
                  <button
                    className="delete-btn"
                    onClick={() => handleDelete(file.id)}
                  >
                    删除
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
