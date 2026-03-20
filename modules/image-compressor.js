/**
 * 图片压缩器 - 使用 Canvas API 压缩图片
 */
(() => {
class ImageCompressor {
  constructor(options = {}) {
    this.maxWidth = options.maxWidth || 1920;
    this.maxHeight = options.maxHeight || 1920;
    this.quality = options.quality || 0.85;
    this.mimeType = options.mimeType || 'image/jpeg';
  }

  // 压缩图片
  async compress(file) {
    return new Promise((resolve, reject) => {
      // 如果是 GIF 或 SVG，不压缩（保持动画或矢量）
      if (file.type === 'image/gif' || file.type === 'image/svg+xml') {
        resolve(file);
        return;
      }

      const reader = new FileReader();

      reader.onerror = () => {
        reject(new Error('文件读取失败'));
      };

      reader.onload = (e) => {
        const img = new Image();

        img.onerror = () => {
          reject(new Error('图片加载失败'));
        };

        img.onload = () => {
          try {
            const canvas = document.createElement('canvas');
            let width = img.width;
            let height = img.height;

            // 计算缩放比例
            let scale = 1;
            if (width > this.maxWidth) {
              scale = this.maxWidth / width;
            }
            if (height > this.maxHeight) {
              scale = Math.min(scale, this.maxHeight / height);
            }

            // 应用缩放
            width = Math.floor(width * scale);
            height = Math.floor(height * scale);

            canvas.width = width;
            canvas.height = height;

            // 绘制图片
            const ctx = canvas.getContext('2d');
            ctx.fillStyle = '#fff'; // 白色背景（针对透明 PNG）
            ctx.fillRect(0, 0, width, height);
            ctx.drawImage(img, 0, 0, width, height);

            // 转为 Blob
            canvas.toBlob(
              (blob) => {
                if (blob) {
                  // 如果压缩后反而更大，使用原文件
                  if (blob.size < file.size) {
                    resolve(blob);
                  } else {
                    console.log('压缩后体积更大，使用原文件');
                    resolve(file);
                  }
                } else {
                  reject(new Error('Canvas toBlob 失败'));
                }
              },
              // PNG 保持 PNG，其他转 JPEG
              file.type === 'image/png' ? 'image/png' : this.mimeType,
              this.quality
            );
          } catch (error) {
            reject(error);
          }
        };

        img.src = e.target.result;
      };

      reader.readAsDataURL(file);
    });
  }

  // 格式化文件大小
  static formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }
}

  window.WechatEditorModules = window.WechatEditorModules || {};
  window.WechatEditorModules.ImageCompressor = ImageCompressor;
})();
