import axios, { AxiosError } from 'axios';
import type {
  UploadResponse,
  MultipartUploadInitResponse,
  ChunkUploadResponse,
  MultipartCompleteResponse,
  UploadStatusResponse,
  FileListResponse,
  ErrorResponse,
  FileInfo,
} from '@/types';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 错误处理
const handleError = (error: unknown): ErrorResponse => {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ErrorResponse>;
    return {
      success: false,
      message: axiosError.response?.data?.message || '请求失败',
      error: axiosError.response?.data?.error || axiosError.message,
    };
  }
  return {
    success: false,
    message: '未知错误',
    error: String(error),
  };
};

// 文件上传服务
export const fileService = {
  /**
   * 单文件上传
   */
  async uploadFile(file: File, onProgress?: (progress: number) => void): Promise<UploadResponse> {
    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await apiClient.post<UploadResponse>('/files/upload', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        onUploadProgress: (progressEvent) => {
          if (progressEvent.total && onProgress) {
            const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total);
            onProgress(progress);
          }
        },
      });
      return response.data;
    } catch (error) {
      throw handleError(error);
    }
  },

  /**
   * 初始化分片上传
   */
  async initMultipartUpload(
    fileName: string,
    fileSize: number,
  ): Promise<MultipartUploadInitResponse> {
    try {
      const response = await apiClient.post<MultipartUploadInitResponse>(
        '/files/upload-multipart/init',
        {
          fileName,
          fileSize,
        },
      );
      return response.data;
    } catch (error) {
      throw handleError(error);
    }
  },

  /**
   * 上传分片
   */
  async uploadChunk(
    uploadId: string,
    chunkIndex: number,
    chunkData: Blob,
  ): Promise<ChunkUploadResponse> {
    const formData = new FormData();
    formData.append('uploadId', uploadId);
    formData.append('chunkIndex', chunkIndex.toString());
    formData.append('chunk', chunkData);

    try {
      const response = await apiClient.post<ChunkUploadResponse>(
        '/files/upload-multipart/chunk',
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        },
      );
      return response.data;
    } catch (error) {
      throw handleError(error);
    }
  },

  /**
   * 完成分片上传
   */
  async completeMultipartUpload(uploadId: string, md5?: string): Promise<MultipartCompleteResponse> {
    try {
      const response = await apiClient.post<MultipartCompleteResponse>(
        '/files/upload-multipart/complete',
        {
          uploadId,
          md5,
        },
      );
      return response.data;
    } catch (error) {
      throw handleError(error);
    }
  },

  /**
   * 获取上传状态
   */
  async getUploadStatus(uploadId: string): Promise<UploadStatusResponse> {
    try {
      const response = await apiClient.get<UploadStatusResponse>(
        `/files/upload-multipart/status/${uploadId}`,
      );
      return response.data;
    } catch (error) {
      throw handleError(error);
    }
  },

  /**
   * 获取文件列表
   */
  async listFiles(): Promise<FileListResponse> {
    try {
      const response = await apiClient.get<FileListResponse>('/files');
      return response.data;
    } catch (error) {
      throw handleError(error);
    }
  },

  /**
   * 下载文件
   */
  downloadFile(fileId: string, fileName: string): void {
    const url = `${API_BASE_URL}/files/${fileId}/download`;
    const link = document.createElement('a');
    link.href = url;
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  },

  /**
   * 健康检查
   */
  async healthCheck(): Promise<{ status: string }> {
    try {
      const response = await apiClient.get<{ status: string }>('/health');
      return response.data;
    } catch (error) {
      throw handleError(error);
    }
  },
};
