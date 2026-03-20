/**
 * 在线编辑器 - 独立页面
 * 基于 app.js 的 STYLES，复用样式系统
 */

const { createApp } = Vue;

const wechatEditorModules = window.WechatEditorModules || {};
const {
  ImageStore,
  ImageCompressor,
  ImageHostManager,
  EditorMethods
} = wechatEditorModules;

const SafeImageStore = ImageStore || class {
  async init() { console.warn('ImageStore 未加载，使用降级实现'); }
  async saveImage() { throw new Error('图片存储模块未加载'); }
  async getImage() { return null; }
  async getImageBlob() { return null; }
  async deleteImage() {}
  async getAllImages() { return []; }
  async clearAll() {}
  async getTotalSize() { return 0; }
};

const SafeImageCompressor = ImageCompressor || class {
  constructor() {}
  async compress(file) { return file; }
  static formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }
};

const SafeImageHostManager = ImageHostManager || class {
  async upload() {
    throw new Error('图床模块未加载');
  }
};

const SafeEditorMethods = EditorMethods || {};

const editorApp = createApp({
  data() {
    return {
      markdownInput: '',
      renderedContent: '',
      currentStyle: 'wechat-default',
      copySuccess: false,
      starredStyles: [],
      toast: {
        show: false,
        message: '',
        type: 'success'
      },
      md: null,
      scanDelimsPatched: false,
      STYLES: STYLES,  // 将样式对象暴露给模板
      turndownService: null,  // Turndown 服务实例
      isDraggingOver: false,  // 拖拽状态
      imageHostManager: new SafeImageHostManager(),  // 图床管理器（已废弃，保留兼容）
      imageStore: null,  // 图片存储管理器（IndexedDB）
      imageCompressor: null,  // 图片压缩器
      imageIdToObjectURL: {},  // 图片 ID 到 Object URL 的映射（用于预览时替换）
      // 小红书相关
      previewMode: 'wechat',  // 预览模式：'wechat' 或 'xiaohongshu'
      xiaohongshuImages: [],  // 生成的小红书图片数组
      xiaohongshuGenerating: false,  // 是否正在生成小红书图片
      // 文章历史记录
      articleHistory: [],           // 历史文章列表
      showHistoryPanel: false,      // 侧边栏显示状态
      currentArticleId: null,       // 当前编辑的文章ID（用于防止重复保存）
      
      // 分享功能
      sharing: false,               // 是否正在分享
      shareUrl: null,               // 分享链接
      shareError: null,             // 分享错误信息
      shareServerUrl: window.location.hostname === 'localhost' 
        ? 'http://localhost:8080' 
        : 'https://md.foolgry.top',  // 根据环境自动选择服务器地址
      shareCopySuccess: false,      // 分享链接复制成功状态
      mermaidInitialized: false     // Mermaid 是否已初始化
    };
  },

  async mounted() {
    // 加载星标样式
    if (typeof this.loadStarredStyles === 'function') {
      this.loadStarredStyles();
    }

    // 加载用户偏好设置
    if (typeof this.loadUserPreferences === 'function') {
      this.loadUserPreferences();
    }

    // 加载文章历史记录
    if (typeof this.loadArticleHistory === 'function') {
      this.loadArticleHistory();
    }

    // 初始化图片存储管理器
    this.imageStore = new SafeImageStore();
    try {
      await this.imageStore.init();
      console.log('图片存储系统已就绪');
    } catch (error) {
      console.error('图片存储系统初始化失败:', error);
      this.showToast('图片存储系统初始化失败', 'error');
    }

    // 初始化图片压缩器（最大宽度 1920px，质量 85%）
    this.imageCompressor = new SafeImageCompressor({
      maxWidth: 1920,
      maxHeight: 1920,
      quality: 0.85
    });

    // 初始化 Turndown 服务（HTML 转 Markdown）
    if (typeof this.initTurndownService === 'function') {
      this.initTurndownService();
    }

    // 初始化 markdown-it
    const md = window.markdownit({
      html: true,
      linkify: true,
      typographer: false,  // 禁用 typographer 以避免智能引号干扰加粗标记
      highlight: function (str, lang) {
        // Mermaid 图表特殊处理
        if (lang && ['mermaid', 'flowchart', 'graph', 'sequenceDiagram', 'gantt', 'classDiagram', 'stateDiagram', 'erDiagram', 'journey', 'pie', 'gitGraph', 'requirementDiagram'].includes(lang)) {
          const mermaidSource = md.utils.escapeHtml(str);
          return `<div class="mermaid" style="background: #fff; padding: 20px; border-radius: 8px; margin: 20px 0; text-align: center; overflow-x: auto;">${mermaidSource}</div>`;
        }

        // macOS 风格的窗口装饰
        const dots = '<div style="display: flex; align-items: center; gap: 6px; padding: 10px 12px; background: #2a2c33; border-bottom: 1px solid #1e1f24;"><span style="width: 12px; height: 12px; border-radius: 50%; background: #ff5f56;"></span><span style="width: 12px; height: 12px; border-radius: 50%; background: #ffbd2e;"></span><span style="width: 12px; height: 12px; border-radius: 50%; background: #27c93f;"></span></div>';

        // 检查 hljs 是否加载
        let codeContent = '';
        if (lang && typeof hljs !== 'undefined') {
          try {
            if (hljs.getLanguage(lang)) {
              codeContent = hljs.highlight(str, { language: lang }).value;
            } else {
              codeContent = md.utils.escapeHtml(str);
            }
          } catch (__) {
            codeContent = md.utils.escapeHtml(str);
          }
        } else {
          codeContent = md.utils.escapeHtml(str);
        }

        return `<div style="margin: 20px 0; border-radius: 8px; overflow: hidden; background: #383a42; box-shadow: 0 2px 8px rgba(0,0,0,0.15);">${dots}<div style="padding: 16px; overflow-x: auto; background: #383a42;"><code style="display: block; color: #abb2bf; font-family: 'SF Mono', Monaco, 'Cascadia Code', Consolas, monospace; font-size: 14px; line-height: 1.6; white-space: pre;">${codeContent}</code></div></div>`;
      }
    });

    if (typeof this.patchMarkdownScanner === 'function') {
      this.patchMarkdownScanner(md);
    }
    this.md = md;

    // 手动触发一次渲染（确保初始内容显示）
    this.$nextTick(() => {
      if (typeof this.renderMarkdown === 'function') {
        this.renderMarkdown();
      }
    });
  },

  watch: {
    currentStyle() {
      if (this.md) {
        if (typeof this.renderMarkdown === 'function') {
          this.renderMarkdown();
        }
      }
      // 保存样式偏好
      if (typeof this.saveUserPreferences === 'function') {
        this.saveUserPreferences();
      }
    },
    markdownInput(newVal, oldVal) {
      if (this.md) {
        if (typeof this.renderMarkdown === 'function') {
          this.renderMarkdown();
        }
      }
      // 自动保存内容（防抖）
      clearTimeout(this._saveTimeout);
      this._saveTimeout = setTimeout(() => {
        if (typeof this.saveUserPreferences === 'function') {
          this.saveUserPreferences();
        }
      }, 1000); // 1秒后保存

      // 当内容被清空时，重置当前文章ID（下次保存会创建新文章）
      if (!newVal || !newVal.trim()) {
        this.currentArticleId = null;
      }
      // 当从空内容粘贴大量内容时，也视为新文章
      else if ((!oldVal || oldVal.trim().length < 10) && newVal.trim().length > 100) {
        this.currentArticleId = null;
      }
    }
  },

  methods: {
    renderMarkdown() {},
    saveUserPreferences() {},
    loadStarredStyles() {},
    loadUserPreferences() {},
    loadArticleHistory() {},
    initTurndownService() {},
    patchMarkdownScanner() {},
    showToast(message, type = 'success') {
      const logger = type === 'error' ? console.error : console.log;
      logger(message);
    },
    ...SafeEditorMethods
  }
});

editorApp.mount('#app');
