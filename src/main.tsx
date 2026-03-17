import React from 'react';
import ReactDOM from 'react-dom/client';
import { FileUploader } from './components/FileUploader';
import './index.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <div className="app">
      <header className="app-header">
        <h1>文件传输服务</h1>
        <p className="subtitle">支持单文件和多文件上传、拖拽上传、断点续传</p>
      </header>
      <main className="app-main">
        <FileUploader />
      </main>
      <footer className="app-footer">
        <p>&copy; 2026 文件传输服务. All rights reserved.</p>
      </footer>
    </div>
  </React.StrictMode>,
);
