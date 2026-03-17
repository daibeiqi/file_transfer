// 上传状态
export type UploadStatus = 'pending' | 'uploading' | 'completed' | 'failed' | 'canceled';

// 上传类型
export type UploadType = 'single' | 'multipart';

// 文件信息
export interface FileInfo {
  id: string;
  originalName: string;
  storedName: string;
  size: number;
  mimeType: string;
  path?: string;
  md5?: string;
  uploadType: UploadType;
  status: UploadStatus;
  createdAt: string;
  updatedAt: string;
}

// 单文件上传响应
export interface UploadResponse {
  success: boolean;
  message: string;
  data?: FileInfo;
}

// 分片上传初始化请求
export interface MultipartUploadInitRequest {
  fileName: string;
  fileSize: number;
  chunkSize?: number;
  contentType?: string;
  md5?: string;
}

// 分片上传初始化响应
export interface MultipartUploadInitResponse {
  success: boolean;
  message: string;
  uploadId?: string;
  chunkSize?: number;
  totalChunks?: number;
}

// 分片上传响应
export interface ChunkUploadResponse {
  success: boolean;
  message: string;
  chunkIndex?: number;
  uploadedChunks?: number[];
}

// 分片上传完成请求
export interface MultipartCompleteRequest {
  uploadId: string;
  md5?: string;
}

// 分片上传完成响应
export interface MultipartCompleteResponse {
  success: boolean;
  message: string;
  data?: FileInfo;
}

// 上传状态响应
export interface UploadStatusResponse {
  success: boolean;
  status: string;
  uploadedSize: number;
  totalSize: number;
  uploadedChunks?: number[];
  totalChunks?: number;
  percentage: number;
}

// 错误响应
export interface ErrorResponse {
  success: boolean;
  message: string;
  error?: string;
}

// 文件列表响应
export interface FileListResponse {
  success: boolean;
  message: string;
  data?: FileInfo[];
  total: number;
}

// 上传进度
export interface UploadProgress {
  fileId: string;
  fileName: string;
  fileSize: number;
  uploadedSize: number;
  totalChunks: number;
  uploadedChunks: number[];
  status: UploadStatus;
  percentage: number;
  speed: number;
  startTime: number;
  lastUpdateTime: number;
}

// 文件上传项
export interface UploadItem {
  id: string;
  file: File;
  progress: UploadProgress;
}
