/**
 * 编辑器方法集合
 */
(() => {
  const wechatEditorModules = window.WechatEditorModules || {};
  const EMPHASIS_MARKERS = wechatEditorModules.EMPHASIS_MARKERS || new Set([0x2A, 0x5F, 0x7E]);
  const isCjkLetter = wechatEditorModules.isCjkLetter || function(charCode) {
    if (!charCode || charCode < 0) {
      return false;
    }

    return (
      (charCode >= 0x3400 && charCode <= 0x4DBF) ||
      (charCode >= 0x4E00 && charCode <= 0x9FFF) ||
      (charCode >= 0xF900 && charCode <= 0xFAFF) ||
      (charCode >= 0xFF01 && charCode <= 0xFF60) ||
      (charCode >= 0xFF61 && charCode <= 0xFF9F) ||
      (charCode >= 0xFFA0 && charCode <= 0xFFDC)
    );
  };

  const ImageCompressor = wechatEditorModules.ImageCompressor || class {
    static formatSize(bytes) {
      if (bytes < 1024) return bytes + ' B';
      if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
      return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    }
  };

  const EditorMethods = {
    loadStarredStyles() {
      try {
        const saved = localStorage.getItem('starredStyles');
        if (saved) {
          this.starredStyles = JSON.parse(saved);
        }
      } catch (error) {
        console.error('加载星标样式失败:', error);
        this.starredStyles = [];
      }
    },

    // 加载用户偏好设置（样式和内容）
    loadUserPreferences() {
      try {
        // 加载样式偏好
        const savedStyle = localStorage.getItem('currentStyle');
        if (savedStyle && STYLES[savedStyle]) {
          this.currentStyle = savedStyle;
        }

        // 加载上次的内容
        const savedContent = localStorage.getItem('markdownInput');
        if (savedContent) {
          this.markdownInput = savedContent;
        } else {
          // 如果没有保存的内容，加载默认示例
          this.loadDefaultExample();
        }
      } catch (error) {
        console.error('加载用户偏好失败:', error);
        // 加载失败时使用默认示例
        this.loadDefaultExample();
      }
    },

    // 保存用户偏好设置
    saveUserPreferences() {
      try {
        // 保存当前样式
        localStorage.setItem('currentStyle', this.currentStyle);

        // 保存当前内容
        localStorage.setItem('markdownInput', this.markdownInput);
      } catch (error) {
        console.error('保存用户偏好失败:', error);
      }
    },

    // 加载默认示例文章
    loadDefaultExample() {
      this.markdownInput = `![](https://images.unsplash.com/photo-1499951360447-b19be8fe80f5?w=1200&h=400&fit=crop)

# 公众号 Markdown 编辑器

欢迎使用这款专为**微信公众号**设计的 Markdown 编辑器！✨

## 🎯 核心功能

### 1. 智能图片处理

![](https://images.unsplash.com/photo-1618005198919-d3d4b5a92ead?w=800&h=500&fit=crop)

- **粘贴即用**：支持从任何地方复制粘贴图片（截图、浏览器、文件管理器）
- **自动压缩**：图片自动压缩，平均压缩 50%-80%
- **本地存储**：使用 IndexedDB 持久化，刷新不丢失
- **编辑流畅**：编辑器中使用短链接，告别卡顿

### 2. 多图排版展示

支持朋友圈式的多图网格布局，2-3 列自动排版：

![](https://images.unsplash.com/photo-1498050108023-c5249f4df085?w=600&h=400&fit=crop)
![](https://images.unsplash.com/photo-1517694712202-14dd9538aa97?w=600&h=400&fit=crop)
![](https://images.unsplash.com/photo-1461749280684-dccba630e2f6?w=600&h=400&fit=crop)

### 3. 13 种精美样式

1. **经典公众号系列**：默认、技术、优雅、深度阅读
2. **传统媒体系列**：杂志、纽约时报、金融时报、Jony Ive
3. **现代数字系列**：Wired、Medium、Apple、Claude、AI Coder

### 4. 一键复制

点击「复制到公众号」按钮，直接粘贴到公众号后台，格式完美保留！

## 💻 代码示例

\`\`\`javascript
// 图片自动压缩并存储到 IndexedDB
const compressedBlob = await imageCompressor.compress(file);
await imageStore.saveImage(imageId, compressedBlob);

// 编辑器中插入短链接
const markdown = \`![图片](img://\${imageId})\`;
\`\`\`

## 📖 引用样式

> 这是一段引用文字，展示编辑器的引用样式效果。
>
> 不同的样式主题会有不同的引用样式，试试切换样式看看效果！

## 📊 表格支持

| 功能 | 支持情况 | 说明 |
|------|---------|------|
| 图片粘贴 | ✅ | 100% 成功率 |
| 刷新保留 | ✅ | IndexedDB 存储 |
| 样式主题 | ✅ | 13 种精选样式 |
| 代码高亮 | ✅ | 多语言支持 |

---

**💡 提示**：

- 试着切换不同的样式主题，体验各种风格的排版效果
- 粘贴图片试试智能压缩功能
- 刷新页面看看内容是否保留

**🌟 开源项目**：如果觉得有用，欢迎访问 [GitHub 仓库](https://github.com/foolgry/editor) 给个 Star！`;
    },

    handleFileUpload(event) {
      const file = event.target.files[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = (e) => {
        this.markdownInput = e.target.result;
      };
      reader.onerror = () => {
        this.showToast('文件读取失败', 'error');
      };
      reader.readAsText(file);

      // 清空 input，允许重复上传同一文件
      event.target.value = '';
    },

    async renderMarkdown() {
      if (!this.markdownInput.trim()) {
        this.renderedContent = '';
        return;
      }

      // 检查 markdown-it 是否已初始化
      if (!this.md) {
        console.warn('markdown-it 尚未初始化，跳过渲染');
        return;
      }

      // 预处理 Markdown
      const processedContent = this.preprocessMarkdown(this.markdownInput);

      // 渲染
      let html = this.md.render(processedContent);

      // 处理 img:// 协议（从 IndexedDB 加载图片）
      html = await this.processImageProtocol(html);

      // 应用样式
      html = this.applyInlineStyles(html);

      this.renderedContent = html;

      // 等待 DOM 更新后渲染 Mermaid
      await this.$nextTick();
      this.renderMermaid();
      if (typeof this.syncPreviewScroll === 'function') {
        this.syncPreviewScroll();
      }
    },

    preprocessMarkdown(content) {
      const renderCore = window.WXMDRenderCore;
      if (renderCore && typeof renderCore.preprocessMarkdown === 'function') {
        return renderCore.preprocessMarkdown(content);
      }

      content = this.stripCitationMarkers(content);

      // 规范化水平分割线格式（修复从飞书等复制时的解析问题）
      // 匹配 * * *、- - -、_ _ _ 等格式（包括带空格的变体）
      // 确保它们被正确解析为 <hr> 而非无序列表
      content = content.replace(/^[ ]{0,3}(\*[ ]*\*[ ]*\*[\* ]*)[ \t]*$/gm, '***');
      content = content.replace(/^[ ]{0,3}(-[ ]*-[ ]*-[- ]*)[ \t]*$/gm, '---');
      content = content.replace(/^[ ]{0,3}(_[ ]*_[ ]*_[_ ]*)[ \t]*$/gm, '___');

      // 修复飞书等复制时的加粗格式断裂问题
      // 例如：**text** **more** -> **text more**（合并相邻的加粗片段）
      // 处理 **空白** (结束后紧跟开始，中间有任意空白) -> 单个空格
      content = content.replace(/\*\*\s+\*\*/g, ' ');
      // 处理 **** 或更多连续星号（通常是格式错误）-> 移除
      content = content.replace(/\*{4,}/g, '');
      // 同样处理下划线格式
      content = content.replace(/__\s+__/g, ' ');
      content = content.replace(/_{4,}/g, '');

      // 规范化列表项格式
      content = content.replace(/^(\s*(?:\d+\.|-|\*)\s+[^:\n]+)\n\s*:\s*(.+?)$/gm, '$1: $2');
      content = content.replace(/^(\s*(?:\d+\.|-|\*)\s+.+?:)\s*\n\s+(.+?)$/gm, '$1 $2');
      content = content.replace(/^(\s*(?:\d+\.|-|\*)\s+[^:\n]+)\n:\s*(.+?)$/gm, '$1: $2');
      content = content.replace(/^(\s*(?:\d+\.|-|\*)\s+.+?)\n\n(\s+.+?)$/gm, (match, listItem, continuation) => {
        const trimmedContinuation = continuation.trimStart();
        // 若续行本身是块级 Markdown（如引用/标题/列表/代码围栏/表格），保持原有空行，不做拼接
        if (/^(>|#{1,6}\s|[-+*]\s|\d+\.\s|```|~~~|\|)/.test(trimmedContinuation)) {
          return match;
        }
        return `${listItem} ${trimmedContinuation}`;
      });

      return content;
    },

    stripCitationMarkers(content) {
      // 过滤 OpenAI/检索链路残留的引用角标，如：citeturn9search6turn9search2
      return content.replace(/\uE200cite\uE202[^\uE201]*\uE201/g, '');
    },

    renderMermaid() {
      if (typeof mermaid === 'undefined') {
        return;
      }

      try {
        if (!this.mermaidInitialized) {
          mermaid.initialize({
            startOnLoad: false,
            theme: 'default',
            securityLevel: 'loose',
            flowchart: {
              useMaxWidth: true,
              htmlLabels: true,
              curve: 'basis'
            },
            sequence: {
              useMaxWidth: true,
              wrap: true
            },
            gantt: {
              useMaxWidth: true
            }
          });
          this.mermaidInitialized = true;
        }

        mermaid.run({
          querySelector: '.preview-container .mermaid'
        });
      } catch (error) {
        console.error('Mermaid 渲染失败:', error);
      }
    },

    // 处理 img:// 协议（从 IndexedDB 加载图片）
    async processImageProtocol(html) {
      if (!this.imageStore) {
        return html;
      }

      // 使用 DOMParser 解析 HTML
      const parser = new DOMParser();
      const doc = parser.parseFromString(html, 'text/html');

      // 查找所有 img 标签
      const images = doc.querySelectorAll('img');

      // 处理每个图片
      for (const img of images) {
        const src = img.getAttribute('src');

        // 检查是否是 img:// 协议
        if (src && src.startsWith('img://')) {
          // 提取图片 ID
          const imageId = src.replace('img://', '');

          try {
            // 从 IndexedDB 获取图片
            let objectURL = this.imageIdToObjectURL[imageId];

            if (!objectURL) {
              // 如果还没有创建 Object URL，现在创建
              objectURL = await this.imageStore.getImage(imageId);

              if (objectURL) {
                // 缓存 Object URL
                this.imageIdToObjectURL[imageId] = objectURL;
              } else {
                console.warn(`图片不存在: ${imageId}`);
                // 图片不存在，显示占位符
                img.setAttribute('src', 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="200" height="200"%3E%3Crect fill="%23ddd" width="200" height="200"/%3E%3Ctext fill="%23999" x="50%25" y="50%25" text-anchor="middle" dy=".3em"%3E图片丢失%3C/text%3E%3C/svg%3E');
                continue;
              }
            }

            // 替换 src 为 Object URL
            img.setAttribute('src', objectURL);

            // 添加 data-image-id 属性（用于复制时识别）
            img.setAttribute('data-image-id', imageId);
          } catch (error) {
            console.error(`加载图片失败 (${imageId}):`, error);
            // 显示错误占位符
            img.setAttribute('src', 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="200" height="200"%3E%3Crect fill="%23fee" width="200" height="200"/%3E%3Ctext fill="%23c00" x="50%25" y="50%25" text-anchor="middle" dy=".3em"%3E加载失败%3C/text%3E%3C/svg%3E');
          }
        }
      }

      return doc.body.innerHTML;
    },

    applyInlineStyles(html) {
      const renderCore = window.WXMDRenderCore;
      if (renderCore && typeof renderCore.applyInlineStyles === 'function') {
        return renderCore.applyInlineStyles(html, {
          styles: STYLES,
          styleKey: this.currentStyle
        });
      }

      const style = STYLES[this.currentStyle].styles;
      const parser = new DOMParser();
      const doc = parser.parseFromString(html, 'text/html');
      const headingInlineOverrides = {
        strong: 'font-weight: 700; color: inherit !important; background-color: transparent !important;',
        em: 'font-style: italic; color: inherit !important; background-color: transparent !important;',
        a: 'color: inherit !important; text-decoration: none !important; border-bottom: 1px solid currentColor !important; background-color: transparent !important;',
        code: 'color: inherit !important; background-color: transparent !important; border: none !important; padding: 0 !important;',
        span: 'color: inherit !important; background-color: transparent !important;',
        b: 'font-weight: 700; color: inherit !important; background-color: transparent !important;',
        i: 'font-style: italic; color: inherit !important; background-color: transparent !important;',
        del: 'color: inherit !important; background-color: transparent !important;',
        mark: 'color: inherit !important; background-color: transparent !important;',
        s: 'color: inherit !important; background-color: transparent !important;',
        u: 'color: inherit !important; text-decoration: underline !important; background-color: transparent !important;',
        ins: 'color: inherit !important; text-decoration: underline !important; background-color: transparent !important;',
        kbd: 'color: inherit !important; background-color: transparent !important; border: none !important; padding: 0 !important;',
        sub: 'color: inherit !important; background-color: transparent !important;',
        sup: 'color: inherit !important; background-color: transparent !important;'
      };
      const headingInlineSelectorList = Object.keys(headingInlineOverrides).join(', ');

      // 先处理图片网格布局（在应用样式之前）
      this.groupConsecutiveImages(doc);

      Object.keys(style).forEach(selector => {
        if (selector === 'pre' || selector === 'code' || selector === 'pre code') {
          return;
        }

        // 跳过已经在网格容器中的图片
        const elements = doc.querySelectorAll(selector);
        elements.forEach(el => {
          // 如果是图片且在网格容器内，跳过样式应用
          if (el.tagName === 'IMG' && el.closest('.image-grid')) {
            return;
          }

          const currentStyle = el.getAttribute('style') || '';
          el.setAttribute('style', currentStyle + '; ' + style[selector]);
        });
      });

      // 标题内的行内元素统一继承标题颜色，避免各主题样式冲突
      const headings = doc.querySelectorAll('h1, h2, h3, h4, h5, h6');
      headings.forEach(heading => {
        const inlineNodes = heading.querySelectorAll(headingInlineSelectorList);
        inlineNodes.forEach(node => {
          const tag = node.tagName.toLowerCase();
          let override = headingInlineOverrides[tag];
          if (!override) {
            return;
          }

          const currentStyle = node.getAttribute('style') || '';
          const sanitizedStyle = currentStyle
            .replace(/color:\s*[^;]+;?/gi, '')
            .replace(/background(?:-color)?:\s*[^;]+;?/gi, '')
            .replace(/border(?:-bottom)?:\s*[^;]+;?/gi, '')
            .replace(/text-decoration:\s*[^;]+;?/gi, '')
            .replace(/box-shadow:\s*[^;]+;?/gi, '')
            .replace(/padding:\s*[^;]+;?/gi, '')
            .replace(/;\s*;/g, ';')
            .trim();
          node.setAttribute('style', sanitizedStyle + '; ' + override);
        });
      });

      const container = doc.createElement('div');
      container.setAttribute('style', style.container);
      container.innerHTML = doc.body.innerHTML;

      return container.outerHTML;
    },

    groupConsecutiveImages(doc) {
      const body = doc.body;
      const children = Array.from(body.children);

      let imagesToProcess = [];

      // 找出所有图片元素，处理两种情况：
      // 1. 多个图片在同一个<p>标签内（连续图片）
      // 2. 每个图片在单独的<p>标签内（分隔的图片）
      children.forEach((child, index) => {
        if (child.tagName === 'P') {
          const images = child.querySelectorAll('img');
          if (images.length > 0) {
            // 如果一个P标签内有多个图片，它们肯定是连续的
            if (images.length > 1) {
              // 多个图片在同一个P标签内，作为一组
              const group = Array.from(images).map(img => ({
                element: child,
                img: img,
                index: index,
                inSameParagraph: true,
                paragraphImageCount: images.length
              }));
              imagesToProcess.push(...group);
            } else if (images.length === 1) {
              // 单个图片在P标签内
              imagesToProcess.push({
                element: child,
                img: images[0],
                index: index,
                inSameParagraph: false,
                paragraphImageCount: 1
              });
            }
          }
        } else if (child.tagName === 'IMG') {
          // 直接是图片元素（少见情况）
          imagesToProcess.push({
            element: child,
            img: child,
            index: index,
            inSameParagraph: false,
            paragraphImageCount: 1
          });
        }
      });

      // 分组逻辑
      let groups = [];
      let currentGroup = [];

      imagesToProcess.forEach((item, i) => {
        if (i === 0) {
          currentGroup.push(item);
        } else {
          const prevItem = imagesToProcess[i - 1];

          // 判断是否连续的条件：
          // 1. 在同一个P标签内的图片肯定是连续的
          // 2. 不同P标签的图片，要看索引是否相邻（差值为1表示相邻）
          let isContinuous = false;

          if (item.index === prevItem.index) {
            // 同一个P标签内的图片
            isContinuous = true;
          } else if (item.index - prevItem.index === 1) {
            // 相邻的P标签，表示连续（没有空行）
            isContinuous = true;
          }
          // 如果索引差大于1，说明中间有其他元素或空行，不连续

          if (isContinuous) {
            currentGroup.push(item);
          } else {
            if (currentGroup.length > 0) {
              groups.push([...currentGroup]);
            }
            currentGroup = [item];
          }
        }
      });

      if (currentGroup.length > 0) {
        groups.push(currentGroup);
      }

      // 对每组图片进行处理
      groups.forEach(group => {
        // 只有2张及以上的图片才需要特殊布局
        if (group.length < 2) return;

        const imageCount = group.length;
        const firstElement = group[0].element;

        // 创建容器
        const gridContainer = doc.createElement('div');
        gridContainer.setAttribute('class', 'image-grid');
        gridContainer.setAttribute('data-image-count', imageCount);

        // 根据图片数量设置网格样式
        let gridStyle = '';
        let columns = 2; // 默认2列

        if (imageCount === 2) {
          gridStyle = `
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 8px;
            margin: 20px auto;
            max-width: 100%;
            align-items: start;
          `;
          columns = 2;
        } else if (imageCount === 3) {
          gridStyle = `
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 8px;
            margin: 20px auto;
            max-width: 100%;
            align-items: start;
          `;
          columns = 3;
        } else if (imageCount === 4) {
          gridStyle = `
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 8px;
            margin: 20px auto;
            max-width: 100%;
            align-items: start;
          `;
          columns = 2;
        } else {
          // 5张及以上，使用3列
          gridStyle = `
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 8px;
            margin: 20px auto;
            max-width: 100%;
            align-items: start;
          `;
          columns = 3;
        }

        gridContainer.setAttribute('style', gridStyle);
        gridContainer.setAttribute('data-columns', columns);

        // 将图片添加到容器中
        group.forEach((item) => {
          const imgWrapper = doc.createElement('div');

          imgWrapper.setAttribute('style', `
            width: 100%;
            height: auto;
            overflow: hidden;
          `);

          const img = item.img.cloneNode(true);
          // 修改图片样式以适应容器，添加圆角
          img.setAttribute('style', `
            width: 100%;
            height: auto;
            display: block;
            border-radius: 8px;
          `.trim());

          imgWrapper.appendChild(img);
          gridContainer.appendChild(imgWrapper);
        });

        // 替换原来的图片元素
        firstElement.parentNode.insertBefore(gridContainer, firstElement);

        // 删除原来的图片元素（需要去重，避免重复删除同一个元素）
        const elementsToRemove = new Set();
        group.forEach(item => {
          elementsToRemove.add(item.element);
        });
        elementsToRemove.forEach(element => {
          if (element.parentNode) {
            element.parentNode.removeChild(element);
          }
        });
      });
    },

    convertGridToTable(doc) {
      // 找到所有的图片网格容器
      const imageGrids = doc.querySelectorAll('.image-grid');

      imageGrids.forEach(grid => {
        // 从data属性获取列数（我们在创建时设置的）
        const columns = parseInt(grid.getAttribute('data-columns')) || 2;
        this.convertToTable(doc, grid, columns);
      });
    },

    convertToTable(doc, grid, columns) {
      // 获取所有图片包装器
      const imgWrappers = Array.from(grid.children);

      // 创建 table 元素
      const table = doc.createElement('table');
      table.setAttribute('style', `
        width: 100% !important;
        border-collapse: collapse !important;
        margin: 20px auto !important;
        table-layout: fixed !important;
        border: none !important;
        background: transparent !important;
      `.trim());

      // 计算需要多少行
      const rows = Math.ceil(imgWrappers.length / columns);

      // 创建表格行
      for (let i = 0; i < rows; i++) {
        const tr = doc.createElement('tr');

        // 创建表格单元格
        for (let j = 0; j < columns; j++) {
          const index = i * columns + j;
          const td = doc.createElement('td');

          td.setAttribute('style', `
            padding: 4px !important;
            vertical-align: top !important;
            width: ${100 / columns}% !important;
            border: none !important;
            background: transparent !important;
          `.trim());

          // 如果有对应的图片，添加到单元格
          if (index < imgWrappers.length) {
            const imgWrapper = imgWrappers[index];
            const img = imgWrapper.querySelector('img');

            if (img) {
              // 根据列数设置不同的图片最大高度 - 确保单行最高360px
              let imgMaxHeight;
              let containerHeight;
              if (columns === 2) {
                imgMaxHeight = '340px';  // 2列布局单张最高340px（留出padding空间）
                containerHeight = '360px';  // 容器高度360px
              } else if (columns === 3) {
                imgMaxHeight = '340px';  // 3列布局单张最高340px
                containerHeight = '360px';  // 容器高度360px
              } else {
                imgMaxHeight = '340px';  // 默认高度340px
                containerHeight = '360px';  // 容器高度360px
              }

              // 创建一个新的包装 div - 添加背景和居中样式（使用table-cell方式，更兼容）
              const wrapper = doc.createElement('div');
              wrapper.setAttribute('style', `
                width: 100% !important;
                height: ${containerHeight} !important;
                text-align: center !important;
                background-color: #f5f5f5 !important;
                border-radius: 4px !important;
                padding: 10px !important;
                box-sizing: border-box !important;
                overflow: hidden !important;
                display: table !important;
              `.trim());

              // 创建内部居中容器
              const innerWrapper = doc.createElement('div');
              innerWrapper.setAttribute('style', `
                display: table-cell !important;
                vertical-align: middle !important;
                text-align: center !important;
              `.trim());

              // 克隆图片并直接设置最大高度
              const newImg = img.cloneNode(true);
              newImg.setAttribute('style', `
                max-width: calc(100% - 20px) !important;
                max-height: ${imgMaxHeight} !important;
                width: auto !important;
                height: auto !important;
                display: inline-block !important;
                margin: 0 auto !important;
                border-radius: 4px !important;
                object-fit: contain !important;
              `.trim());

              innerWrapper.appendChild(newImg);
              wrapper.appendChild(innerWrapper);
              td.appendChild(wrapper);
            }
          }

          tr.appendChild(td);
        }

        table.appendChild(tr);
      }

      // 替换网格为 table
      grid.parentNode.replaceChild(table, grid);
    },

    async copyToClipboard() {
      if (!this.renderedContent) {
        this.showToast('没有内容可复制', 'error');
        return;
      }

      try {
        const parser = new DOMParser();
        const doc = parser.parseFromString(this.renderedContent, 'text/html');

        // 将图片网格转换为 table 布局（公众号兼容）
        this.convertGridToTable(doc);

        // 处理图片：转为 Base64
        const images = doc.querySelectorAll('img');
        if (images.length > 0) {
          this.showToast(`正在处理 ${images.length} 张图片...`, 'success');

          let successCount = 0;
          let failCount = 0;

          const imagePromises = Array.from(images).map(async (img) => {
            try {
              const base64 = await this.convertImageToBase64(img);
              img.setAttribute('src', base64);
              successCount++;
            } catch (error) {
              console.error('图片转换失败:', img.getAttribute('src'), error);
              failCount++;
              // 失败时保持原URL
            }
          });

          await Promise.all(imagePromises);

          if (failCount > 0) {
            this.showToast(`图片处理完成：${successCount} 成功，${failCount} 失败（保留原链接）`, 'error');
          }
        }

        // Section 容器包裹
        const styleConfig = STYLES[this.currentStyle];
        const containerBg = this.extractBackgroundColor(styleConfig.styles.container);

        if (containerBg && containerBg !== '#fff' && containerBg !== '#ffffff') {
          const section = doc.createElement('section');
          const containerStyle = styleConfig.styles.container;
          const paddingMatch = containerStyle.match(/padding:\s*([^;]+)/);
          const maxWidthMatch = containerStyle.match(/max-width:\s*([^;]+)/);
          const padding = paddingMatch ? paddingMatch[1].trim() : '40px 20px';
          const maxWidth = maxWidthMatch ? maxWidthMatch[1].trim() : '100%';

          section.setAttribute('style',
            `background-color: ${containerBg}; ` +
            `padding: ${padding}; ` +
            `max-width: ${maxWidth}; ` +
            `margin: 0 auto; ` +
            `box-sizing: border-box; ` +
            `word-wrap: break-word;`
          );

          while (doc.body.firstChild) {
            section.appendChild(doc.body.firstChild);
          }

          const allElements = section.querySelectorAll('*');
          allElements.forEach(el => {
            const currentStyle = el.getAttribute('style') || '';
            let newStyle = currentStyle;
            newStyle = newStyle.replace(/max-width:\s*[^;]+;?/g, '');
            newStyle = newStyle.replace(/margin:\s*0\s+auto;?/g, '');
            if (newStyle.includes(`background-color: ${containerBg}`)) {
              newStyle = newStyle.replace(new RegExp(`background-color:\\s*${containerBg.replace(/[()]/g, '\\$&')};?`, 'g'), '');
            }
            newStyle = newStyle.replace(/;\s*;/g, ';').replace(/^\s*;\s*|\s*;\s*$/g, '').trim();
            if (newStyle) {
              el.setAttribute('style', newStyle);
            } else {
              el.removeAttribute('style');
            }
          });

          doc.body.appendChild(section);
        }

        // 代码块简化
        const codeBlocks = doc.querySelectorAll('div[style*="border-radius: 8px"]');
        codeBlocks.forEach(block => {
          const codeElement = block.querySelector('code');
          if (codeElement) {
            const codeText = codeElement.textContent || codeElement.innerText;
            const pre = doc.createElement('pre');
            const code = doc.createElement('code');

            pre.setAttribute('style',
              'background: linear-gradient(to bottom, #2a2c33 0%, #383a42 8px, #383a42 100%);' +
              'padding: 0;' +
              'border-radius: 6px;' +
              'overflow: hidden;' +
              'margin: 24px 0;' +
              'box-shadow: 0 2px 8px rgba(0,0,0,0.15);'
            );

            code.setAttribute('style',
              'color: #abb2bf;' +
              'font-family: "SF Mono", Consolas, Monaco, "Courier New", monospace;' +
              'font-size: 14px;' +
              'line-height: 1.7;' +
              'display: block;' +
              'white-space: pre;' +
              'padding: 16px 20px;' +
              '-webkit-font-smoothing: antialiased;' +
              '-moz-osx-font-smoothing: grayscale;'
            );

            code.textContent = codeText;
            pre.appendChild(code);
            block.parentNode.replaceChild(pre, block);
          }
        });

        // 列表项扁平化
        const listItems = doc.querySelectorAll('li');
        listItems.forEach(li => {
          let text = li.textContent || li.innerText;
          text = text.replace(/\n/g, ' ').replace(/\r/g, ' ').replace(/\s+/g, ' ').trim();
          li.innerHTML = '';
          li.textContent = text;
          const currentStyle = li.getAttribute('style') || '';
          li.setAttribute('style', currentStyle);
        });

        // 深色模式适配：调整引用块样式，使用透明黑色让微信自动转换
        const blockquotes = doc.querySelectorAll('blockquote');
        blockquotes.forEach(blockquote => {
          const currentStyle = blockquote.getAttribute('style') || '';

          // 移除现有的背景色和文字颜色
          let newStyle = currentStyle
            .replace(/background(?:-color)?:\s*[^;]+;?/gi, '')
            .replace(/color:\s*[^;]+;?/gi, '');

          // 添加深色模式友好的样式
          // 使用半透明黑色背景和文字，微信会在深色模式下自动反转
          newStyle += '; background: rgba(0, 0, 0, 0.05) !important';
          newStyle += '; color: rgba(0, 0, 0, 0.8) !important';

          // 清理多余的分号
          newStyle = newStyle.replace(/;\s*;/g, ';').replace(/^\s*;\s*|\s*;\s*$/g, '').trim();
          blockquote.setAttribute('style', newStyle);
        });

        const simplifiedHTML = doc.body.innerHTML;
        const plainText = doc.body.textContent || '';

        const htmlBlob = new Blob([simplifiedHTML], { type: 'text/html' });
        const textBlob = new Blob([plainText], { type: 'text/plain' });

        const clipboardItem = new ClipboardItem({
          'text/html': htmlBlob,
          'text/plain': textBlob
        });

        await navigator.clipboard.write([clipboardItem]);

        this.copySuccess = true;
        this.showToast('复制成功', 'success');

        // 自动保存到历史记录
        this.saveToHistory();

        setTimeout(() => {
          this.copySuccess = false;
        }, 2000);
      } catch (error) {
        console.error('复制失败:', error);
        this.showToast('复制失败', 'error');
      }
    },

    async convertImageToBase64(imgElement) {
      const src = imgElement.getAttribute('src');

      // 如果已经是Base64，直接返回
      if (src.startsWith('data:')) {
        return src;
      }

      // 优先处理：检查是否有 data-image-id（来自 IndexedDB）
      const imageId = imgElement.getAttribute('data-image-id');
      if (imageId && this.imageStore) {
        try {
          // 从 IndexedDB 获取图片 Blob
          const blob = await this.imageStore.getImageBlob(imageId);

          if (blob) {
            // 将 Blob 转为 Base64
            return new Promise((resolve, reject) => {
              const reader = new FileReader();
              reader.onloadend = () => resolve(reader.result);
              reader.onerror = (error) => reject(new Error('FileReader failed: ' + error));
              reader.readAsDataURL(blob);
            });
          } else {
            console.warn(`图片 Blob 不存在: ${imageId}`);
            // 继续尝试用 fetch 方式（兜底）
          }
        } catch (error) {
          console.error(`从 IndexedDB 读取图片失败 (${imageId}):`, error);
          // 继续尝试用 fetch 方式（兜底）
        }
      }

      // 后备方案：尝试通过 URL 获取图片
      try {
        const response = await fetch(src, {
          mode: 'cors',
          cache: 'default'
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const blob = await response.blob();

        return new Promise((resolve, reject) => {
          const reader = new FileReader();
          reader.onloadend = () => resolve(reader.result);
          reader.onerror = (error) => reject(new Error('FileReader failed: ' + error));
          reader.readAsDataURL(blob);
        });
      } catch (error) {
        // CORS或网络错误时，抛出错误让外层处理
        throw new Error(`图片加载失败 (${src}): ${error.message}`);
      }
    },

    extractBackgroundColor(styleString) {
      if (!styleString) return null;

      const bgColorMatch = styleString.match(/background-color:\s*([^;]+)/);
      if (bgColorMatch) {
        return bgColorMatch[1].trim();
      }

      const bgMatch = styleString.match(/background:\s*([#rgb][^;]+)/);
      if (bgMatch) {
        const bgValue = bgMatch[1].trim();
        if (bgValue.startsWith('#') || bgValue.startsWith('rgb')) {
          return bgValue;
        }
      }

      return null;
    },

    isStyleStarred(styleKey) {
      return this.starredStyles.includes(styleKey);
    },

    isRecommended(styleKey) {
      // 推荐的样式
      const recommended = ['nikkei', 'wechat-anthropic', 'wechat-ft', 'wechat-nyt', 'latepost-depth', 'wechat-tech'];
      return recommended.includes(styleKey);
    },

    toggleStarStyle(styleKey) {
      const index = this.starredStyles.indexOf(styleKey);
      if (index > -1) {
        this.starredStyles.splice(index, 1);
        this.showToast('已取消收藏', 'success');
      } else {
        this.starredStyles.push(styleKey);
        this.showToast('已收藏样式', 'success');
      }
      this.saveStarredStyles();
    },

    saveStarredStyles() {
      try {
        localStorage.setItem('starredStyles', JSON.stringify(this.starredStyles));
      } catch (error) {
        console.error('保存星标样式失败:', error);
      }
    },

    getStyleName(styleKey) {
      const style = STYLES[styleKey];
      return style ? style.name : styleKey;
    },

    showToast(message, type = 'success') {
      this.toast.show = true;
      this.toast.message = message;
      this.toast.type = type;

      setTimeout(() => {
        this.toast.show = false;
      }, 3000);
    },

    patchMarkdownScanner(md) {
      const renderCore = window.WXMDRenderCore;
      if (renderCore && typeof renderCore.patchMarkdownScanner === 'function') {
        renderCore.patchMarkdownScanner(md);
        this.scanDelimsPatched = true;
        return;
      }

      if (!md || !md.inline || !md.inline.State || this.scanDelimsPatched) {
        return;
      }

      const utils = md.utils;
      const StateInline = md.inline.State;
      const allowLeadingPunctuation = this.createSafeLeadingPunctuationChecker();

      const originalScanDelims = StateInline.prototype.scanDelims;

      StateInline.prototype.scanDelims = function (start, canSplitWord) {
        const max = this.posMax;
        const marker = this.src.charCodeAt(start);

        if (!EMPHASIS_MARKERS.has(marker)) {
          return originalScanDelims.call(this, start, canSplitWord);
        }

        const lastChar = start > 0 ? this.src.charCodeAt(start - 1) : 0x20;

        let pos = start;
        while (pos < max && this.src.charCodeAt(pos) === marker) {
          pos++;
        }

        const count = pos - start;
        const nextChar = pos < max ? this.src.charCodeAt(pos) : 0x20;

        const isLastWhiteSpace = utils.isWhiteSpace(lastChar);
        const isNextWhiteSpace = utils.isWhiteSpace(nextChar);

        let isLastPunctChar =
          utils.isMdAsciiPunct(lastChar) || utils.isPunctChar(String.fromCharCode(lastChar));

        let isNextPunctChar =
          utils.isMdAsciiPunct(nextChar) || utils.isPunctChar(String.fromCharCode(nextChar));

        if (isNextPunctChar && allowLeadingPunctuation(nextChar, marker)) {
          isNextPunctChar = false;
        }

        if (marker === 0x5F /* _ */) {
          if (!isLastWhiteSpace && !isLastPunctChar && isCjkLetter(lastChar)) {
            isLastPunctChar = true;
          }
          if (!isNextWhiteSpace && !isNextPunctChar && isCjkLetter(nextChar)) {
            isNextPunctChar = true;
          }
        }

        const left_flanking =
          !isNextWhiteSpace && (!isNextPunctChar || isLastWhiteSpace || isLastPunctChar);
        const right_flanking =
          !isLastWhiteSpace && (!isLastPunctChar || isNextWhiteSpace || isNextPunctChar);

        const can_open = left_flanking && (canSplitWord || !right_flanking || isLastPunctChar);
        const can_close = right_flanking && (canSplitWord || !left_flanking || isNextPunctChar);

        return { can_open, can_close, length: count };
      };

      this.scanDelimsPatched = true;
    },

    createSafeLeadingPunctuationChecker() {
      const fallbackChars = '「『《〈（【〔〖［｛﹁﹃﹙﹛﹝“‘（';
      const fallbackSet = new Set(
        fallbackChars.split('').map(char => char.codePointAt(0))
      );

      let unicodeRegex = null;
      try {
        unicodeRegex = new RegExp('[\\p{Ps}\\p{Pi}]', 'u');
      } catch (_error) {
        unicodeRegex = null;
      }

      return (charCode, marker) => {
        if (!EMPHASIS_MARKERS.has(marker)) {
          return false;
        }

        if (unicodeRegex) {
          const char = String.fromCharCode(charCode);
          if (unicodeRegex.test(char)) {
            return true;
          }
        }

        return fallbackSet.has(charCode);
      };
    },

    // 初始化 Turndown 服务
    initTurndownService() {
      if (typeof TurndownService === 'undefined') {
        console.warn('Turndown 库未加载，智能粘贴功能将不可用');
        return;
      }

      this.turndownService = new TurndownService({
        headingStyle: 'atx',        // 使用 # 样式的标题
        bulletListMarker: '-',       // 无序列表使用 -
        codeBlockStyle: 'fenced',    // 代码块使用 ```
        fence: '```',                // 代码块围栏
        emDelimiter: '*',            // 斜体使用 *
        strongDelimiter: '**',       // 加粗使用 **
        linkStyle: 'inlined'         // 链接使用内联样式
      });

      // 配置表格支持
      this.turndownService.keep(['table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td']);

      // 自定义规则：保留表格结构
      this.turndownService.addRule('table', {
        filter: 'table',
        replacement: (_content, node) => {
          // 简单的表格转换为 Markdown 表格
          const rows = Array.from(node.querySelectorAll('tr'));
          if (rows.length === 0) return '';

          let markdown = '\n\n';
          let headerProcessed = false;

          rows.forEach((row, index) => {
            const cells = Array.from(row.querySelectorAll('td, th'));
            const cellContents = cells.map(cell => {
              // 清理单元格内容
              const text = cell.textContent.replace(/\n/g, ' ').trim();
              return text;
            });

            if (cellContents.length > 0) {
              markdown += '| ' + cellContents.join(' | ') + ' |\n';

              // 第一行后添加分隔符
              if (index === 0 || (!headerProcessed && row.querySelector('th'))) {
                markdown += '| ' + cells.map(() => '---').join(' | ') + ' |\n';
                headerProcessed = true;
              }
            }
          });

          return markdown + '\n';
        }
      });

      // 自定义规则：优化图片处理
      this.turndownService.addRule('image', {
        filter: 'img',
        replacement: (_content, node) => {
          const alt = node.alt || '图片';
          const src = node.src || '';
          const title = node.title || '';

          // 处理 base64 图片（截取前30个字符作为标识）
          if (src.startsWith('data:image')) {
            const type = src.match(/data:image\/(\w+);/)?.[1] || 'image';
            return `![${alt}](data:image/${type};base64,...)${title ? ` "${title}"` : ''}\n`;
          }

          return `![${alt}](${src})${title ? ` "${title}"` : ''}\n`;
        }
      });
    },

    // 处理粘贴事件
    async handleSmartPaste(event) {
      console.log('===== handleSmartPaste 被调用 =====');

      const clipboardData = event.clipboardData || event.originalEvent?.clipboardData;

      if (!clipboardData) {
        console.log('不支持 clipboardData');
        return; // 不支持的浏览器，使用默认行为
      }

      // 调试模式（需要时可以打开）
      const DEBUG = true;
      if (DEBUG) {
        console.log('剪贴板数据类型:', Array.from(clipboardData.types || []));
      }

      // 检查是否有文件（某些应用复制图片会作为文件）
      if (clipboardData.files && clipboardData.files.length > 0) {
        if (DEBUG) console.log('检测到文件:', clipboardData.files[0]);
        const file = clipboardData.files[0];
        if (file && file.type && file.type.startsWith('image/')) {
          event.preventDefault();
          await this.handleImageUpload(file, event.target);
          return;
        }
      }

      // 检查 items（浏览器复制的图片通常在这里）
      const items = clipboardData.items;
      if (items) {
        for (let item of items) {
          if (DEBUG) console.log('Item 类型:', item.type, 'Kind:', item.kind);

          // 检查是否是图片
          if (item.kind === 'file' && item.type && item.type.indexOf('image') !== -1) {
            event.preventDefault();
            const file = item.getAsFile();
            if (file) {
              await this.handleImageUpload(file, event.target);
              return; // 处理完图片就返回
            }
          }
        }
      }

      // 获取剪贴板中的各种格式数据
      const htmlData = clipboardData.getData('text/html');
      const textData = clipboardData.getData('text/plain');

      // 检查是否是类似 [Image #2] 这样的占位符文本
      if (textData && /^\[Image\s*#?\d*\]$/i.test(textData.trim())) {
        if (DEBUG) console.warn('检测到图片占位符文本，但无法获取实际图片数据');
        this.showToast('⚠️ 请尝试：截图工具 / 浏览器复制 / 拖拽文件', 'error');
        event.preventDefault();
        return; // 不插入占位符文本
      }

      if (DEBUG) {
        console.log('纯文本数据:', textData?.substring(0, 200));
        console.log('HTML 数据:', htmlData?.substring(0, 200));
        console.log('是否检测为 Markdown:', textData && this.isMarkdown(textData));
        console.log('是否有 turndownService:', !!this.turndownService);
      }

      // 检查是否来自 IDE/代码编辑器的 HTML（需要特殊处理）
      const isFromIDE = this.isIDEFormattedHTML(htmlData, textData);

      if (DEBUG) {
        console.log('是否来自 IDE:', isFromIDE);
      }

      if (isFromIDE && textData && this.isMarkdown(textData)) {
        // 来自 IDE 的 Markdown 代码，直接使用纯文本（避免转义）
        if (DEBUG) console.log('检测到 IDE 复制的 Markdown 代码，使用纯文本');
        return; // 使用默认粘贴行为
      }

      // 处理 HTML 数据（富文本编辑器或其他来源）
      if (htmlData && htmlData.trim() !== '' && this.turndownService) {
        // 检查是否是从代码编辑器复制的（精确匹配真正的代码块标签，避免误判）
        // 只有当 HTML 主要由 <pre> 或 <code> 组成时才跳过转换
        const hasPreTag = /<pre[\s>]/.test(htmlData);
        const hasCodeTag = /<code[\s>]/.test(htmlData);
        const isMainlyCode = (hasPreTag || hasCodeTag) && !htmlData.includes('<p') && !htmlData.includes('<div');

        if (isMainlyCode) {
          // 真正的代码编辑器内容，使用纯文本
          if (DEBUG) console.log('检测到代码编辑器格式，使用纯文本');
          return; // 使用默认粘贴行为
        }

        // 检查 HTML 中是否包含本地文件路径的图片（如 file:/// 协议）
        if (htmlData.includes('file:///') || htmlData.includes('src="file:')) {
          if (DEBUG) console.warn('检测到本地文件路径的图片，无法直接上传');
          this.showToast('⚠️ 本地图片请直接拖拽文件到编辑器', 'error');
          event.preventDefault();
          return;
        }

        event.preventDefault(); // 阻止默认粘贴

        try {
          // 将 HTML 转换为 Markdown
          let markdown = this.turndownService.turndown(htmlData);

          // 清理多余的空行
          markdown = markdown.replace(/\n{3,}/g, '\n\n');

          // 获取当前光标位置
          const textarea = event.target;
          const start = textarea.selectionStart;
          const end = textarea.selectionEnd;
          const value = textarea.value;

          // 插入转换后的 Markdown
          const newValue = value.substring(0, start) + markdown + value.substring(end);

          // 更新文本框内容
          this.markdownInput = newValue;

          // 恢复光标位置
          this.$nextTick(() => {
            textarea.selectionStart = textarea.selectionEnd = start + markdown.length;
            textarea.focus();
          });

          // 显示提示
          this.showToast('✨ 已智能转换为 Markdown 格式', 'success');
        } catch (error) {
          if (DEBUG) console.error('HTML 转 Markdown 失败:', error);
          // 转换失败，使用纯文本
          this.insertTextAtCursor(event.target, textData);
        }
      }
      // 检查纯文本是否为 Markdown（后备方案，只有在没有 HTML 时才检查）
      else if (textData && this.isMarkdown(textData)) {
        // 已经是 Markdown，直接使用纯文本
        if (DEBUG) console.log('没有 HTML，但检测到 Markdown 格式，使用纯文本');
        return; // 使用默认粘贴行为
      }
      // 普通文本，使用默认粘贴行为
      else {
        if (DEBUG) console.log('普通文本，使用默认粘贴行为');
        return; // 使用默认行为
      }
    },

    // 检测文本是否为 Markdown 格式
    isMarkdown(text) {
      if (!text) return false;

      // Markdown 特征模式
      const patterns = [
        /^#{1,6}\s+/m,           // 标题
        /\*\*[^*]+\*\*/,         // 加粗
        /\*[^*\n]+\*/,           // 斜体
        /\[[^\]]+\]\([^)]+\)/,   // 链接
        /!\[[^\]]*\]\([^)]+\)/,  // 图片
        /^[\*\-\+]\s+/m,         // 无序列表
        /^\d+\.\s+/m,            // 有序列表
        /^>\s+/m,                // 引用
        /`[^`]+`/,               // 内联代码
        /```[\s\S]*?```/,        // 代码块
        /^\|.*\|$/m,             // 表格
        /<!--.*?-->/,            // HTML 注释（我们的图片注释）
        /^---+$/m                // 分隔线
      ];

      // 计算匹配的特征数量
      const matchCount = patterns.filter(pattern => pattern.test(text)).length;

      // 如果有 2 个或以上的 Markdown 特征，认为是 Markdown
      // 或者如果包含我们的图片注释，也认为是 Markdown
      return matchCount >= 2 || text.includes('<!-- img:');
    },

    // 检测 HTML 是否来自 IDE/代码编辑器
    isIDEFormattedHTML(htmlData, textData) {
      if (!htmlData || !textData) return false;

      // IDE 复制的 HTML 特征（VS Code、Cursor、Sublime Text 等）
      const ideSignatures = [
        // VS Code 特征
        /<meta\s+charset=['"]utf-8['"]/i,
        /<div\s+class=["']ace_line["']/,
        /style=["'][^"']*font-family:\s*['"]?(?:Consolas|Monaco|Menlo|Courier)/i,

        // 简单的 div/span 结构（没有富文本语义标签）
        // 检查：有 HTML 标签，但几乎没有 <p>, <h1-h6>, <strong>, <em> 等富文本标签
        function(html) {
          const hasDivSpan = /<(?:div|span)[\s>]/.test(html);
          const hasSemanticTags = /<(?:p|h[1-6]|strong|em|ul|ol|li|blockquote)[\s>]/i.test(html);
          // 如果有 div/span 但几乎没有语义标签，可能是代码编辑器
          return hasDivSpan && !hasSemanticTags;
        },

        // 检查 HTML 是否只是简单包裹纯文本（几乎没有格式化）
        function(html) {
          // 去除所有 HTML 标签，看是否与纯文本几乎一致
          const strippedHtml = html.replace(/<[^>]+>/g, '').trim();
          const similarity = strippedHtml === textData.trim();
          return similarity;
        }
      ];

      // 检查是否匹配任何 IDE 特征
      let matchCount = 0;
      for (const signature of ideSignatures) {
        if (typeof signature === 'function') {
          if (signature(htmlData)) matchCount++;
        } else if (signature.test(htmlData)) {
          matchCount++;
        }
      }

      // 如果匹配 2 个或以上特征，认为是 IDE 格式
      return matchCount >= 2;
    },

    // 在光标位置插入文本
    insertTextAtCursor(textarea, text) {
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      const value = textarea.value;

      const newValue = value.substring(0, start) + text + value.substring(end);
      this.markdownInput = newValue;

      this.$nextTick(() => {
        textarea.selectionStart = textarea.selectionEnd = start + text.length;
        textarea.focus();
      });
    },

    // 处理图片上传 - 压缩并存储到 IndexedDB
    async handleImageUpload(file, textarea) {
      // 检查文件类型
      if (!file.type.startsWith('image/')) {
        this.showToast('请上传图片文件', 'error');
        return;
      }

      // 检查文件大小（10MB限制）
      const maxSize = 10 * 1024 * 1024;
      if (file.size > maxSize) {
        this.showToast('图片大小不能超过 10MB', 'error');
        return;
      }

      const imageName = file.name.replace(/\.[^/.]+$/, '') || '图片';
      const originalSize = file.size;

      try {
        // 第一步：压缩图片
        this.showToast('🔄 正在压缩图片...', 'success');

        const compressedBlob = await this.imageCompressor.compress(file);
        const compressedSize = compressedBlob.size;

        // 计算压缩率
        const compressionRatio = ((1 - compressedSize / originalSize) * 100).toFixed(0);
        console.log(`图片压缩完成: ${ImageCompressor.formatSize(originalSize)} → ${ImageCompressor.formatSize(compressedSize)} (压缩 ${compressionRatio}%)`);

        // 第二步：生成唯一 ID
        const imageId = `img-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

        // 第三步：存储到 IndexedDB
        await this.imageStore.saveImage(imageId, compressedBlob, {
          name: imageName,
          originalName: file.name,
          originalSize: originalSize,
          compressedSize: compressedSize,
          compressionRatio: compressionRatio,
          mimeType: compressedBlob.type || file.type
        });

        // 第四步：插入 img:// 协议的短链接到编辑器
        const markdownImage = `![${imageName}](img://${imageId})`;

        if (textarea) {
          const currentPos = textarea.selectionStart;
          const before = this.markdownInput.substring(0, currentPos);
          const after = this.markdownInput.substring(currentPos);

          this.markdownInput = before + markdownImage + after;

          this.$nextTick(() => {
            const newPos = currentPos + markdownImage.length;
            textarea.selectionStart = textarea.selectionEnd = newPos;
            textarea.focus();
          });
        } else {
          this.markdownInput += '\n' + markdownImage;
        }

        // 第五步：显示成功提示
        if (compressionRatio > 10) {
          this.showToast(`✅ 已保存 (${ImageCompressor.formatSize(originalSize)} → ${ImageCompressor.formatSize(compressedSize)})`, 'success');
        } else {
          this.showToast(`✅ 已保存 (${ImageCompressor.formatSize(compressedSize)})`, 'success');
        }
      } catch (error) {
        console.error('图片处理失败:', error);
        this.showToast('❌ 图片处理失败: ' + error.message, 'error');
      }
    },

    // 处理文件拖拽
    handleDrop(event) {
      event.preventDefault();
      event.stopPropagation();

      this.isDraggingOver = false;

      const files = event.dataTransfer.files;
      if (files.length > 0) {
        const file = files[0];
        if (file.type.startsWith('image/')) {
          this.handleImageUpload(file, event.target);
        } else {
          this.showToast('只支持拖拽图片文件', 'error');
        }
      }
    },

    // 阻止默认拖拽行为
    handleDragOver(event) {
      event.preventDefault();
      event.stopPropagation();
      event.dataTransfer.dropEffect = 'copy';
      this.isDraggingOver = true;
    },

    // 处理拖拽进入
    handleDragEnter(event) {
      event.preventDefault();
      this.isDraggingOver = true;
    },

    // 处理拖拽离开
    handleDragLeave(event) {
      event.preventDefault();
      // 只有当真正离开编辑器时才移除状态
      if (event.target.classList.contains('markdown-input')) {
        this.isDraggingOver = false;
      }
    },

    // ============ 小红书功能相关方法 ============

    // 生成小红书图片
    async generateXiaohongshuImages() {
      if (!this.renderedContent) {
        this.showToast('没有内容可生成', 'error');
        return;
      }

      if (typeof html2canvas === 'undefined') {
        this.showToast('html2canvas 库未加载', 'error');
        return;
      }

      this.xiaohongshuGenerating = true;
      this.xiaohongshuImages = [];

      try {
        // 创建临时渲染容器
        const tempContainer = this.createXiaohongshuContainer();
        document.body.appendChild(tempContainer);

        // 计算文章信息
        const articleInfo = this.calculateArticleInfo();

        // 分页
        const pages = await this.splitContentIntoPages(tempContainer, articleInfo);

        if (pages.length === 0) {
          throw new Error('内容为空，无法生成图片');
        }

        // 生成每一页的图片
        for (let i = 0; i < pages.length; i++) {
          const pageElement = pages[i];

          // 添加页码
          this.addPageNumber(pageElement, i + 1, pages.length);

          // 如果是首页，添加信息面板
          if (i === 0) {
            this.addInfoPanel(pageElement, articleInfo);
          }

          // 将页面元素添加到容器中，确保 html2canvas 可以找到它
          tempContainer.appendChild(pageElement);

          // 等待一小段时间确保元素渲染完成
          await new Promise(resolve => setTimeout(resolve, 100));

          // 生成图片
          const canvas = await html2canvas(pageElement, {
            scale: 2,
            useCORS: true,
            allowTaint: true,
            backgroundColor: this.getBackgroundColor(),
            width: 750,
            height: 1000,
            windowWidth: 750,
            windowHeight: 1000,
            logging: false
          });

          const dataUrl = canvas.toDataURL('image/png');
          this.xiaohongshuImages.push({
            dataUrl: dataUrl,
            pageNumber: i + 1,
            totalPages: pages.length
          });

          // 移除页面元素，准备下一页
          tempContainer.removeChild(pageElement);
        }

        // 清理临时容器
        document.body.removeChild(tempContainer);

        this.showToast(`成功生成 ${pages.length} 张小红书图片`, 'success');
      } catch (error) {
        console.error('生成小红书图片失败:', error);
        this.showToast('生成失败: ' + error.message, 'error');

        // 确保清理临时容器
        const existingContainer = document.querySelector('div[style*="-9999px"]');
        if (existingContainer) {
          document.body.removeChild(existingContainer);
        }
      } finally {
        this.xiaohongshuGenerating = false;
      }
    },

    // 创建小红书渲染容器
    createXiaohongshuContainer() {
      const container = document.createElement('div');
      container.style.position = 'fixed';
      container.style.left = '-9999px';
      container.style.top = '0';
      container.style.width = '750px';
      container.style.pointerEvents = 'none';
      container.style.zIndex = '-1';
      // 不设置 visibility: hidden，因为 html2canvas 需要可见元素
      return container;
    },

    // 计算文章信息
    calculateArticleInfo() {
      const parser = new DOMParser();
      const doc = parser.parseFromString(this.renderedContent, 'text/html');

      // 计算字数（去除HTML标签）
      const textContent = doc.body.textContent || '';
      const charCount = textContent.replace(/\s/g, '').length;

      // 计算阅读时长（假设每分钟阅读400字）
      const readingTime = Math.ceil(charCount / 400);

      // 计算图片数量
      const imageCount = doc.querySelectorAll('img').length;

      return {
        charCount,
        readingTime,
        imageCount
      };
    },

    // 分页算法 - 完全简化版本
    async splitContentIntoPages(container, articleInfo) {
      // 解析 Markdown 为纯文本结构（不使用复杂的渲染样式）
      const simplifiedContent = this.createSimplifiedContent();

      const pages = [];
      const maxPageHeight = 850; // 留出空间给页码和首页信息面板

      // 创建测量容器
      const measureContainer = this.createPageElement();
      container.appendChild(measureContainer);

      let currentPageContent = [];
      let currentHeight = 0;
      const firstPageOffset = 120; // 首页信息面板占用空间

      for (let i = 0; i < simplifiedContent.length; i++) {
        const block = simplifiedContent[i];

        // 创建元素
        const element = this.createSimplifiedElement(block);

        // 添加到测量容器
        measureContainer.appendChild(element);
        const elementHeight = element.offsetHeight || 50;

        // 计算是否超出页面高度
        const heightLimit = pages.length === 0 ? maxPageHeight - firstPageOffset : maxPageHeight;
        const wouldExceed = currentHeight + elementHeight > heightLimit;

        if (wouldExceed && currentPageContent.length > 0) {
          // 创建新页面
          const page = this.createPageElement();
          currentPageContent.forEach(el => page.appendChild(el));
          pages.push(page);

          currentPageContent = [];
          currentHeight = 0;
        }

        // 从测量容器移除
        measureContainer.removeChild(element);
        currentPageContent.push(element);
        currentHeight += elementHeight;
      }

      // 添加最后一页
      if (currentPageContent.length > 0) {
        const page = this.createPageElement();
        currentPageContent.forEach(el => page.appendChild(el));
        pages.push(page);
      }

      // 清理测量容器
      container.removeChild(measureContainer);

      return pages;
    },

    // 创建简化的内容结构（纯文本，无复杂样式）
    createSimplifiedContent() {
      const lines = this.markdownInput.split('\n');
      const content = [];

      lines.forEach(line => {
        line = line.trim();
        if (!line) return;

        // 标题
        if (line.startsWith('# ')) {
          content.push({ type: 'h1', text: line.substring(2) });
        } else if (line.startsWith('## ')) {
          content.push({ type: 'h2', text: line.substring(3) });
        } else if (line.startsWith('### ')) {
          content.push({ type: 'h3', text: line.substring(4) });
        }
        // 列表
        else if (line.startsWith('- ') || line.startsWith('* ')) {
          content.push({ type: 'li', text: line.substring(2) });
        }
        // 引用
        else if (line.startsWith('> ')) {
          content.push({ type: 'quote', text: line.substring(2) });
        }
        // 代码块标记（跳过）
        else if (line.startsWith('```')) {
          // 跳过代码块
        }
        // 图片（跳过，小红书图片由外链显示）
        else if (line.startsWith('![')) {
          // 跳过图片
        }
        // 分隔线
        else if (line === '---') {
          content.push({ type: 'hr' });
        }
        // 普通段落
        else {
          // 移除 Markdown 格式标记
          let text = line.replace(/\*\*(.+?)\*\*/g, '$1'); // 粗体
          text = text.replace(/\*(.+?)\*/g, '$1'); // 斜体
          text = text.replace(/`(.+?)`/g, '$1'); // 行内代码
          content.push({ type: 'p', text: text });
        }
      });

      return content;
    },

    // 创建简化的元素（只使用基本的内联样式）
    createSimplifiedElement(block) {
      const el = document.createElement('div');

      switch (block.type) {
        case 'h1':
          el.textContent = block.text;
          el.style.fontSize = '28px';
          el.style.fontWeight = 'bold';
          el.style.margin = '20px 0 10px 0';
          el.style.color = '#000';
          break;
        case 'h2':
          el.textContent = block.text;
          el.style.fontSize = '24px';
          el.style.fontWeight = 'bold';
          el.style.margin = '16px 0 8px 0';
          el.style.color = '#000';
          break;
        case 'h3':
          el.textContent = block.text;
          el.style.fontSize = '20px';
          el.style.fontWeight = 'bold';
          el.style.margin = '12px 0 6px 0';
          el.style.color = '#333';
          break;
        case 'p':
          el.textContent = block.text;
          el.style.fontSize = '16px';
          el.style.lineHeight = '1.8';
          el.style.margin = '8px 0';
          el.style.color = '#333';
          break;
        case 'li':
          el.textContent = '• ' + block.text;
          el.style.fontSize = '16px';
          el.style.lineHeight = '1.8';
          el.style.margin = '4px 0';
          el.style.paddingLeft = '10px';
          el.style.color = '#333';
          break;
        case 'quote':
          el.textContent = block.text;
          el.style.fontSize = '15px';
          el.style.lineHeight = '1.8';
          el.style.margin = '8px 0';
          el.style.padding = '10px 15px';
          el.style.borderLeft = '3px solid #0066FF';
          el.style.background = '#f5f5f5';
          el.style.color = '#666';
          break;
        case 'hr':
          el.style.height = '1px';
          el.style.background = '#ddd';
          el.style.margin = '20px 0';
          el.style.border = 'none';
          break;
      }

      return el;
    },

    // 创建页面元素
    createPageElement() {
      const page = document.createElement('div');
      page.style.width = '750px';
      page.style.height = '1000px';
      page.style.backgroundColor = this.getBackgroundColor();
      page.style.padding = '80px 40px 40px 40px';
      page.style.boxSizing = 'border-box';
      page.style.position = 'relative';
      page.style.overflow = 'hidden';
      page.style.fontFamily = 'Arial';
      page.style.fontSize = '16px';
      page.style.lineHeight = '1.8';
      page.style.color = '#333';
      return page;
    },

    // 添加页码
    addPageNumber(pageElement, currentPage, totalPages) {
      const pageNumber = document.createElement('div');
      pageNumber.textContent = `${currentPage}/${totalPages}`;
      pageNumber.style.position = 'absolute';
      pageNumber.style.bottom = '30px';
      pageNumber.style.right = '40px';
      pageNumber.style.fontSize = '14px';
      pageNumber.style.color = '#999';
      pageNumber.style.fontWeight = '500';
      pageElement.appendChild(pageNumber);
    },

    // 添加首页信息面板
    addInfoPanel(pageElement, articleInfo) {
      const panel = document.createElement('div');
      panel.style.position = 'absolute';
      panel.style.top = '20px';
      panel.style.left = '40px';
      panel.style.right = '40px';
      panel.style.padding = '20px';
      panel.style.backgroundColor = '#E6F0FF';
      panel.style.borderRadius = '8px';
      panel.style.border = '1px solid #99CCFF';

      const infoItems = [
        { label: '字数', value: articleInfo.charCount },
        { label: '阅读', value: `${articleInfo.readingTime}分钟` },
        { label: '图片', value: `${articleInfo.imageCount}张` }
      ];

      // 创建容器（使用 table 布局）
      const table = document.createElement('table');
      table.style.width = '100%';
      table.style.borderCollapse = 'collapse';
      const tr = document.createElement('tr');

      infoItems.forEach(item => {
        const td = document.createElement('td');
        td.style.textAlign = 'center';
        td.style.padding = '5px';

        const valueDiv = document.createElement('div');
        valueDiv.textContent = item.value;
        valueDiv.style.fontSize = '24px';
        valueDiv.style.fontWeight = 'bold';
        valueDiv.style.color = '#0066FF';
        valueDiv.style.marginBottom = '4px';

        const labelDiv = document.createElement('div');
        labelDiv.textContent = item.label;
        labelDiv.style.fontSize = '12px';
        labelDiv.style.color = '#666';

        td.appendChild(valueDiv);
        td.appendChild(labelDiv);
        tr.appendChild(td);
      });

      table.appendChild(tr);
      panel.appendChild(table);

      // 插入到页面顶部
      pageElement.insertBefore(panel, pageElement.firstChild);
    },

    // 获取背景色
    getBackgroundColor() {
      const styleConfig = STYLES[this.currentStyle];
      if (styleConfig && styleConfig.styles && styleConfig.styles.container) {
        const bgColor = this.extractBackgroundColor(styleConfig.styles.container);
        return bgColor || '#FFFFFF';
      }
      return '#FFFFFF';
    },

    // 下载单张小红书图片
    downloadXiaohongshuImage(image, index) {
      const link = document.createElement('a');
      link.download = `小红书-第${index + 1}张-共${this.xiaohongshuImages.length}张.png`;
      link.href = image.dataUrl;
      link.click();
      this.showToast(`下载第 ${index + 1} 张图片`, 'success');
    },

    // 批量下载小红书图片
    async downloadAllXiaohongshuImages() {
      if (this.xiaohongshuImages.length === 0) {
        this.showToast('没有图片可下载', 'error');
        return;
      }

      this.showToast(`开始下载 ${this.xiaohongshuImages.length} 张图片...`, 'success');

      for (let i = 0; i < this.xiaohongshuImages.length; i++) {
        const image = this.xiaohongshuImages[i];

        // 添加延迟，避免浏览器阻止批量下载
        await new Promise(resolve => setTimeout(resolve, 300));

        const link = document.createElement('a');
        link.download = `小红书-第${i + 1}张-共${this.xiaohongshuImages.length}张.png`;
        link.href = image.dataUrl;
        link.click();
      }

      this.showToast('批量下载完成', 'success');
    },

    // ==================== 文章历史记录功能 ====================

    // 从 Markdown 内容提取标题
    extractTitle(markdownContent) {
      if (!markdownContent || !markdownContent.trim()) {
        return '无标题';
      }

      // 尝试匹配第一个 # 标题
      const titleMatch = markdownContent.match(/^#\s+(.+)$/m);
      if (titleMatch && titleMatch[1]) {
        // 清理标题中的 markdown 格式
        let title = titleMatch[1].trim();
        title = title.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, '');
        return title.substring(0, 50); // 最多 50 字符
      }

      // 如果没有标题，取前 20 个字符
      const cleanContent = markdownContent
        .replace(/^!\[.*?\]\(.*?\)$/gm, '') // 移除图片
        .replace(/^#+\s*/gm, '') // 移除标题标记
        .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1') // 移除链接格式
        .replace(/[*_~`]/g, '') // 移除格式标记
        .trim();

      if (cleanContent) {
        return cleanContent.substring(0, 20) + (cleanContent.length > 20 ? '...' : '');
      }

      return '无标题';
    },

    // 保存当前文章到历史记录
    saveToHistory() {
      const content = this.markdownInput;
      if (!content || !content.trim()) {
        this.showToast('内容为空，无法保存', 'error');
        return;
      }

      const title = this.extractTitle(content);
      const now = Date.now();

      // 如果有当前文章ID，直接更新该文章
      if (this.currentArticleId) {
        const existingIndex = this.articleHistory.findIndex(
          article => article.id === this.currentArticleId
        );

        if (existingIndex !== -1) {
          // 更新已存在的文章
          this.articleHistory[existingIndex].title = title;
          this.articleHistory[existingIndex].content = content;
          this.articleHistory[existingIndex].style = this.currentStyle;
          this.articleHistory[existingIndex].updatedAt = now;

          // 移到最前面
          const article = this.articleHistory.splice(existingIndex, 1)[0];
          this.articleHistory.unshift(article);

          this.saveArticleHistory();
          this.showToast('已更新历史记录', 'success');
          return;
        }
      }

      // 没有当前文章ID，创建新文章
      const newArticleId = `article-${now}-${Math.random().toString(36).substring(2, 8)}`;
      const newArticle = {
        id: newArticleId,
        title: title,
        content: content,
        style: this.currentStyle,
        createdAt: now,
        updatedAt: now
      };

      // 添加到列表开头
      this.articleHistory.unshift(newArticle);

      // 设置为当前文章
      this.currentArticleId = newArticleId;

      // 限制最多 20 篇
      if (this.articleHistory.length > 20) {
        this.articleHistory = this.articleHistory.slice(0, 20);
      }

      // 保存到 localStorage
      this.saveArticleHistory();
      this.showToast('已保存到历史记录', 'success');
    },

    // 从历史记录加载文章
    loadFromHistory(articleId) {
      const article = this.articleHistory.find(a => a.id === articleId);
      if (!article) {
        this.showToast('文章不存在', 'error');
        return;
      }

      // 恢复内容和样式
      this.markdownInput = article.content;
      if (article.style && STYLES[article.style]) {
        this.currentStyle = article.style;
      }

      // 设置当前文章ID，后续编辑会更新这篇文章
      this.currentArticleId = articleId;

      // 关闭侧边栏
      this.showHistoryPanel = false;

      this.showToast('已加载文章', 'success');
    },

    // 从历史记录删除文章
    deleteFromHistory(articleId) {
      const index = this.articleHistory.findIndex(a => a.id === articleId);
      if (index === -1) {
        this.showToast('文章不存在', 'error');
        return;
      }

      this.articleHistory.splice(index, 1);
      this.saveArticleHistory();
      this.showToast('已删除', 'success');
    },

    // 从 localStorage 加载历史记录
    loadArticleHistory() {
      try {
        const saved = localStorage.getItem('articleHistory');
        if (saved) {
          const data = JSON.parse(saved);
          if (data && Array.isArray(data.articles)) {
            this.articleHistory = data.articles;
          }
        }
      } catch (error) {
        console.error('加载历史记录失败:', error);
        this.articleHistory = [];
      }
    },

    // 保存历史记录到 localStorage
    saveArticleHistory() {
      try {
        const data = {
          articles: this.articleHistory
        };
        localStorage.setItem('articleHistory', JSON.stringify(data));
      } catch (error) {
        console.error('保存历史记录失败:', error);
        this.showToast('保存历史记录失败', 'error');
      }
    },

    // 切换历史记录侧边栏
    toggleHistoryPanel() {
      this.showHistoryPanel = !this.showHistoryPanel;
    },

    // 格式化历史记录时间显示
    formatHistoryDate(timestamp) {
      if (!timestamp) return '';

      const date = new Date(timestamp);
      const now = new Date();
      const diff = now - date;

      // 不到 1 分钟
      if (diff < 60 * 1000) {
        return '刚刚';
      }

      // 不到 1 小时
      if (diff < 60 * 60 * 1000) {
        const minutes = Math.floor(diff / (60 * 1000));
        return `${minutes} 分钟前`;
      }

      // 不到 24 小时
      if (diff < 24 * 60 * 60 * 1000) {
        const hours = Math.floor(diff / (60 * 60 * 1000));
        return `${hours} 小时前`;
      }

      // 今年内
      const year = date.getFullYear();
      const month = String(date.getMonth() + 1).padStart(2, '0');
      const day = String(date.getDate()).padStart(2, '0');
      const hour = String(date.getHours()).padStart(2, '0');
      const minute = String(date.getMinutes()).padStart(2, '0');

      if (year === now.getFullYear()) {
        return `${month}-${day} ${hour}:${minute}`;
      }

      // 往年
      return `${year}-${month}-${day}`;
    },

    // ==================== 分享功能 ====================

    // 分享内容
    async shareContent() {
      if (!this.markdownInput || !this.markdownInput.trim()) {
        this.showToast('内容为空，无法分享', 'error');
        return;
      }

      this.sharing = true;
      this.shareError = null;

      try {
        const shareContent = await this.prepareShareContent();
        const response = await fetch(`${this.shareServerUrl}/api/share`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            content: shareContent,
            style: this.currentStyle
          })
        });

        const data = await response.json();

        if (!response.ok) {
          throw new Error(data.error || '分享失败');
        }

        this.shareUrl = `${this.shareServerUrl}/s/${data.id}`;
        this.showToast('分享成功！链接已生成', 'success');

        // 复制到剪贴板
        await this.copyShareUrl();
      } catch (error) {
        console.error('分享失败:', error);
        this.shareError = error.message;
        this.showToast(`分享失败: ${error.message}`, 'error');
      } finally {
        this.sharing = false;
      }
    },

    async prepareShareContent() {
      const imagePattern = /!\[([^\]]*)\]\(img:\/\/([^)]+)\)/g;
      const matches = Array.from(this.markdownInput.matchAll(imagePattern));

      if (!matches.length || !this.imageStore) {
        return this.markdownInput;
      }

      const replacements = await Promise.all(matches.map(async (match) => {
        const [markdownImage, altText, imageId] = match;

        try {
          const blob = await this.imageStore.getImageBlob(imageId);
          if (!blob) {
            return [markdownImage, markdownImage];
          }

          const dataUrl = await this.blobToDataUrl(blob);
          return [markdownImage, `![${altText}](${dataUrl})`];
        } catch (error) {
          console.error(`分享图片转换失败 (${imageId}):`, error);
          return [markdownImage, markdownImage];
        }
      }));

      return replacements.reduce((content, [from, to]) => {
        return content.split(from).join(to);
      }, this.markdownInput);
    },

    blobToDataUrl(blob) {
      return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(reader.result);
        reader.onerror = () => reject(reader.error || new Error('图片读取失败'));
        reader.readAsDataURL(blob);
      });
    },

    // 复制分享链接
    async copyShareUrl() {
      if (!this.shareUrl) return;

      try {
        await navigator.clipboard.writeText(this.shareUrl);
        this.shareCopySuccess = true;
        this.showToast('链接已复制到剪贴板', 'success');

        setTimeout(() => {
          this.shareCopySuccess = false;
        }, 2000);
      } catch (err) {
        console.error('复制链接失败:', err);
        // 降级方案
        const textarea = document.createElement('textarea');
        textarea.value = this.shareUrl;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        this.shareCopySuccess = true;
        this.showToast('链接已复制到剪贴板', 'success');

        setTimeout(() => {
          this.shareCopySuccess = false;
        }, 2000);
      }
    },

    // ==================== 主题管理功能 ====================

    // 从 localStorage 加载隐藏的主题配置
    loadHiddenStyles() {
      try {
        const saved = localStorage.getItem('hiddenStyles');
        if (saved) {
          this.hiddenStyles = JSON.parse(saved);
        }
      } catch (error) {
        console.error('加载隐藏主题配置失败:', error);
        this.hiddenStyles = [];
      }
    },

    // 保存隐藏的主题配置到 localStorage
    saveHiddenStyles() {
      try {
        localStorage.setItem('hiddenStyles', JSON.stringify(this.hiddenStyles));
      } catch (error) {
        console.error('保存隐藏主题配置失败:', error);
      }
    },

    // 切换主题的显示/隐藏状态
    toggleStyleVisibility(styleKey) {
      const index = this.hiddenStyles.indexOf(styleKey);
      if (index > -1) {
        // 取消隐藏
        this.hiddenStyles.splice(index, 1);
        this.showToast(`已显示「${this.getStyleName(styleKey)}」`, 'success');
      } else {
        // 隐藏主题
        this.hiddenStyles.push(styleKey);

        // 如果隐藏的是当前正在使用的主题，自动切换到第一个可用主题
        if (styleKey === this.currentStyle) {
          this.switchToFirstAvailableStyle();
        }

        this.showToast(`已隐藏「${this.getStyleName(styleKey)}」`, 'success');
      }
    },

    // 重置所有主题为显示状态
    resetHiddenStyles() {
      this.hiddenStyles = [];
      this.showToast('已重置为默认，所有主题都显示', 'success');
    },

    // 检查主题是否被隐藏
    isStyleHidden(styleKey) {
      return this.hiddenStyles.includes(styleKey);
    },

    // 切换到第一个可用的主题
    switchToFirstAvailableStyle() {
      // 获取所有可见的主题（未被隐藏的）
      const visibleStyles = Object.keys(STYLES).filter(key => !this.isStyleHidden(key));

      if (visibleStyles.length > 0) {
        // 切换到第一个可见主题
        this.currentStyle = visibleStyles[0];
        this.showToast(`已自动切换到「${this.getStyleName(this.currentStyle)}」`, 'success');
      } else {
        // 如果没有可见主题，显示第一个主题（防止页面异常）
        const firstKey = Object.keys(STYLES)[0];
        if (firstKey) {
          this.currentStyle = firstKey;
          // 从隐藏列表中移除这个主题
          const index = this.hiddenStyles.indexOf(firstKey);
          if (index > -1) {
            this.hiddenStyles.splice(index, 1);
          }
          this.showToast(`所有主题都被隐藏，已恢复「${this.getStyleName(firstKey)}」`, 'error');
        }
      }
    },

    // 切换主题管理弹窗的显示状态
    toggleSettingsModal() {
      this.showSettingsModal = !this.showSettingsModal;
    },

    // ==================== 主题顺序管理 ====================

    // 从 localStorage 加载主题顺序
    loadStyleOrder() {
      try {
        const saved = localStorage.getItem('styleOrder');
        if (saved) {
          const order = JSON.parse(saved);
          if (Array.isArray(order)) {
            // 过滤掉已删除的主题，保留有效key的顺序
            const validKeys = order.filter(key => STYLES[key]);
            // 获取新增的主题（在当前顺序中不存在的key）
            const newKeys = Object.keys(STYLES).filter(key => !validKeys.includes(key));
            // 合并：保留已有顺序，追加新增主题
            this.styleOrder = [...validKeys, ...newKeys];
          } else {
            this.styleOrder = Object.keys(STYLES);
          }
        } else {
          // 默认使用 STYLES 中的顺序
          this.styleOrder = Object.keys(STYLES);
        }
      } catch (error) {
        console.error('加载主题顺序失败:', error);
        this.styleOrder = Object.keys(STYLES);
      }
    },

    // 保存主题顺序到 localStorage
    saveStyleOrder() {
      try {
        localStorage.setItem('styleOrder', JSON.stringify(this.styleOrder));
      } catch (error) {
        console.error('保存主题顺序失败:', error);
      }
    },

    // 获取按顺序排列的主题列表
    getOrderedStyles() {
      // 如果有自定义顺序，使用它；否则使用默认顺序
      if (this.styleOrder && this.styleOrder.length > 0) {
        // 过滤掉不存在的主题（可能后续删除了）
        return this.styleOrder.filter(key => STYLES[key]);
      }
      return Object.keys(STYLES);
    },

    // 获取可见主题数量
    getVisibleStylesCount() {
      return this.getOrderedStyles().filter(key => !this.isStyleHidden(key)).length;
    },

    // 获取设置弹窗用的可见主题列表（用于响应式更新）
    getVisibleStylesForSettings() {
      return this.getOrderedStyles().filter(key => !this.isStyleHidden(key));
    },

    // 获取设置弹窗用的隐藏主题列表（用于响应式更新）
    getHiddenStylesForSettings() {
      return this.getOrderedStyles().filter(key => this.isStyleHidden(key));
    },

    // 恢复默认排序
    resetStyleOrder() {
      // 只重置顺序，保留隐藏状态
      this.styleOrder = Object.keys(STYLES);
      this.showToast('已恢复默认排序', 'success');
    },

    // 将主题移动到隐藏区域
    moveToHidden(styleKey) {
      if (!this.hiddenStyles.includes(styleKey)) {
        this.hiddenStyles.push(styleKey);
        // 如果隐藏的是当前主题，自动切换
        if (styleKey === this.currentStyle) {
          this.switchToFirstAvailableStyle();
        }
        this.showToast(`已隐藏「${this.getStyleName(styleKey)}」`, 'success');
      }
    },

    // 将主题移动到可见区域
    moveToVisible(styleKey) {
      const index = this.hiddenStyles.indexOf(styleKey);
      if (index > -1) {
        this.hiddenStyles.splice(index, 1);
        this.showToast(`已显示「${this.getStyleName(styleKey)}」`, 'success');
      }
    },

    // ==================== 拖拽功能 ====================

    // 开始拖拽
    handleDragStart(event, styleKey, sourceZone) {
      this.draggingStyle = styleKey;
      this.dragSourceZone = sourceZone;
      event.dataTransfer.effectAllowed = 'move';
      // 设置拖拽数据
      event.dataTransfer.setData('text/plain', styleKey);
    },

    // 结束拖拽
    handleDragEnd() {
      this.draggingStyle = null;
      this.dragSourceZone = null;
    },

    // 处理放置
    handleDrop(event, targetZone) {
      event.preventDefault();
      const styleKey = event.dataTransfer.getData('text/plain');
      if (!styleKey || !STYLES[styleKey]) return;

      const sourceZone = this.dragSourceZone;

      if (targetZone === 'hidden') {
        // 拖放到隐藏区域
        this.moveToHidden(styleKey);
        // 将该主题移动到顺序列表末尾（以便下次显示时位置合理）
        this.moveStyleToEnd(styleKey);
      } else if (targetZone === 'visible') {
        // 拖放到可见区域
        if (sourceZone === 'hidden') {
          // 从隐藏区域拖过来
          this.moveToVisible(styleKey);
        }
        // 计算放置位置并重新排序
        this.handleReorder(event, styleKey);
      }
    },

    // 处理重排序
    handleReorder(event, draggedKey) {
      // 获取鼠标位置下方的元素（使用改进的算法）
      const dragZone = event.currentTarget;
      const { afterElement, insertAtStart } = this.getDropPosition(dragZone, event.clientX, event.clientY);

      // 获取当前可见主题的顺序
      const visibleKeys = this.getOrderedStyles().filter(key => !this.isStyleHidden(key));

      // 移除被拖拽的元素（从可见列表中）
      const draggedIndex = visibleKeys.indexOf(draggedKey);
      if (draggedIndex > -1) {
        visibleKeys.splice(draggedIndex, 1);
      }

      // 确定插入位置
      let insertIndex;
      if (insertAtStart) {
        // 放到最前面
        insertIndex = 0;
      } else if (afterElement) {
        // 放到指定元素后面
        const afterKey = afterElement.dataset.key;
        const afterIndex = visibleKeys.indexOf(afterKey);
        if (afterIndex > -1) {
          insertIndex = afterIndex + 1;
        } else {
          insertIndex = visibleKeys.length;
        }
      } else {
        // 放到末尾
        insertIndex = visibleKeys.length;
      }

      // 插入到新位置
      visibleKeys.splice(insertIndex, 0, draggedKey);

      // 将隐藏的主题追加到末尾，保持它们原有的相对顺序
      const hiddenKeys = this.getOrderedStyles().filter(key => this.isStyleHidden(key));

      // 合并可见和隐藏的主题
      const newOrder = [...visibleKeys, ...hiddenKeys];

      // 更新顺序
      this.styleOrder = newOrder;
    },

    // 获取拖拽放置位置（支持横向和纵向布局）
    getDropPosition(container, x, y) {
      const draggableElements = [...container.querySelectorAll('.style-card-drag:not(.dragging)')];

      if (draggableElements.length === 0) {
        return { afterElement: null, insertAtStart: true };
      }

      // 获取容器边界
      const containerRect = container.getBoundingClientRect();

      // 检查是否在最前面（鼠标在第一个元素之前）
      const firstElement = draggableElements[0];
      const firstRect = firstElement.getBoundingClientRect();

      // 判断鼠标是否在第一个元素的左上方
      if (y < firstRect.bottom && x < firstRect.left + firstRect.width / 2) {
        return { afterElement: null, insertAtStart: true };
      }

      // 遍历所有元素，找到鼠标位置后的元素
      let closestElement = null;
      let closestDistance = Infinity;

      for (const element of draggableElements) {
        const rect = element.getBoundingClientRect();

        // 计算元素中心点
        const centerX = rect.left + rect.width / 2;
        const centerY = rect.top + rect.height / 2;

        // 计算鼠标到元素中心点的距离
        const distance = Math.sqrt(Math.pow(x - centerX, 2) + Math.pow(y - centerY, 2));

        // 只考虑鼠标前方的元素
        const isAfterMouse = x < centerX || (x < rect.right && y < centerY);

        if (isAfterMouse && distance < closestDistance) {
          closestDistance = distance;
          closestElement = element;
        }
      }

      return { afterElement: closestElement, insertAtStart: false };
    },

    // 将主题移动到顺序列表末尾
    moveStyleToEnd(styleKey) {
      const currentOrder = [...this.getOrderedStyles()];
      const index = currentOrder.indexOf(styleKey);
      if (index > -1) {
        currentOrder.splice(index, 1);
        currentOrder.push(styleKey);
        this.styleOrder = currentOrder;
      }
    },

    // ==================== 右键菜单功能 ====================

    // 显示主题右键菜单
    showStyleContextMenu(event, styleKey) {
      // 阻止默认右键菜单
      event.preventDefault();

      // 设置目标主题
      this.contextMenuTargetStyle = styleKey;

      // 计算菜单位置（确保不超出屏幕）
      const menuWidth = 160;
      const menuHeight = 120; // 估计高度
      let x = event.clientX;
      let y = event.clientY;

      // 检查右边界
      if (x + menuWidth > window.innerWidth) {
        x = window.innerWidth - menuWidth - 10;
      }

      // 检查下边界
      if (y + menuHeight > window.innerHeight) {
        y = window.innerHeight - menuHeight - 10;
      }

      this.contextMenuPosition = { x, y };
      this.showContextMenu = true;
    },

    // 关闭右键菜单
    closeContextMenu() {
      this.showContextMenu = false;
      this.contextMenuTargetStyle = null;
    },

    // 处理右键菜单操作
    handleContextMenuAction(action) {
      const styleKey = this.contextMenuTargetStyle;
      if (!styleKey) return;

      switch (action) {
        case 'toggle-star':
          this.toggleStarStyle(styleKey);
          break;
        case 'toggle-visibility':
          this.toggleStyleVisibility(styleKey);
          break;
        case 'open-settings':
          this.showSettingsModal = true;
          break;
      }

      // 关闭菜单
      this.closeContextMenu();
    }
  };

  window.WechatEditorModules = window.WechatEditorModules || {};
  window.WechatEditorModules.EditorMethods = EditorMethods;
})();
