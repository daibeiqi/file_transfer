# 文件传输服务

一个支持单文件和多文件上传、拖拽上传、断点续传的文件传输系统。

## 功能特性

- 支持单文件和多文件上传
- 支持拖拽上传和点击选择
- 显示上传进度
- 文件类型和大小限制（最大 100MB）
- 支持断点续传（分片上传）
- 文件下载功能
- 文件列表查看

## 技术栈

### 后端
- Go 1.25+
- Gin Web Framework
- Multipart 文件处理

### 前端
- React 18
- TypeScript
- Vite
- Axios

### 存储
- 本地文件系统

## 项目结构

```
git_file_transfer/
├── internal/
│   ├── api/
│   │   ├── file_types.go      # API 类型定义
│   │   └── routes.go          # 路由定义
│   ├── storage/
│   │   └── file_manager.go    # 文件管理器
│   └── types/
│       └── upload.go          # 上传类型
├── src/
│   ├── components/
│   │   └── FileUploader.tsx   # 文件上传组件
│   ├── services/
│   │   └── api.ts             # API 服务
│   └── types/
│       └── index.ts           # 类型定义
├── uploads/                   # 上传文件存储目录
├── main.go                    # 后端入口文件
└── index.html                 # 前端入口文件
```

## 快速开始

### 后端运行

```bash
# 编译并运行后端
go run main.go
```

后端服务将在 `http://localhost:8080` 启动。

### 前端运行

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

前端开发服务器将在 `http://localhost:3000` 启动。

### 构建

```bash
# 构建前端
npm run build

# 构建后端
go build -o file-transfer
```

## API 端点

### 单文件上传
- `POST /api/files/upload` - 上传单个文件

### 分片上传
- `POST /api/files/upload-multipart/init` - 初始化分片上传
- `POST /api/files/upload-multipart/chunk` - 上传分片
- `POST /api/files/upload-multipart/complete` - 完成分片上传
- `GET /api/files/upload-multipart/status/:uploadId` - 获取上传状态

### 文件操作
- `GET /api/files` - 获取文件列表
- `GET /api/files/:id/download` - 下载文件

## 配置

### 文件大小限制
- 默认最大文件大小: 100MB
- 可在 `internal/storage/file_manager.go` 中修改 `MaxFileSize` 常量

### 分片大小
- 默认分片大小: 5MB
- 可在 `internal/storage/file_manager.go` 中修改 `DefaultChunkSize` 常量

### 支持的文件类型
默认支持以下文件类型:
- 图片: .jpg, .jpeg, .png, .gif, .webp, .svg
- 文档: .pdf, .doc, .docx, .xls, .xlsx, .ppt, .pptx, .txt
- 压缩包: .zip, .rar, .7z, .tar, .gz

可在 `internal/storage/file_manager.go` 中修改 `AllowedExtensions` 变量。

## 开发

### 代码规范
- Go 遵循 `gofmt` 和 `golint` 规范
- TypeScript 使用 ESLint 进行代码检查

### 运行测试
```bash
# 后端测试
go test ./...

# 前端测试
npm test
```

## License

MIT License
