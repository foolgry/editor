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
      shareCopySuccess: false,      // 分享链接复制成功状态
      shareServerUrl: window.location.origin, // 分享服务器地址
      mermaidInitialized: false,    // Mermaid 是否已初始化

      // 主题管理
      hiddenStyles: [],             // 被隐藏的主题 ID 数组
      showSettingsModal: false,     // 主题管理弹窗显示状态
      styleOrder: [],               // 主题显示顺序（为空时使用默认顺序）

      // 拖拽状态
      draggingStyle: null,          // 正在拖拽的主题
      dragSourceZone: null,         // 拖拽来源区域

      // 右键菜单
      showContextMenu: false,         // 是否显示右键菜单
      contextMenuPosition: { x: 0, y: 0 }, // 右键菜单位置
      contextMenuTargetStyle: null    // 右键菜单针对的主题
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

    // 加载隐藏的主题配置
    if (typeof this.loadHiddenStyles === 'function') {
      this.loadHiddenStyles();
    }

    // 加载主题顺序配置
    if (typeof this.loadStyleOrder === 'function') {
      this.loadStyleOrder();
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

    // 初始化 markdown-it（统一渲染内核）
    const renderCore = window.WXMDRenderCore;
    if (renderCore && typeof renderCore.createMarkdownParser === 'function') {
      this.md = renderCore.createMarkdownParser({
        markdownit: window.markdownit,
        hljs: typeof hljs !== 'undefined' ? hljs : null
      });
    } else {
      // 降级兼容：当共享渲染内核未加载时，仍可使用基础 markdown-it
      this.md = window.markdownit({
        html: true,
        linkify: true,
        typographer: false
      });
      if (typeof this.patchMarkdownScanner === 'function') {
        this.patchMarkdownScanner(this.md);
      }
    }

    // 手动触发一次渲染（确保初始内容显示）
    this.$nextTick(() => {
      if (typeof this.renderMarkdown === 'function') {
        this.renderMarkdown();
      }
      if (typeof this.bindEditorScrollSync === 'function') {
        this.bindEditorScrollSync();
      }
    });

    // 监听键盘事件（ESC 关闭右键菜单）
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape' && this.showContextMenu) {
        this.showContextMenu = false;
      }
    });
  },

  beforeUnmount() {
    if (typeof this.unbindEditorScrollSync === 'function') {
      this.unbindEditorScrollSync();
    }
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
    hiddenStyles: {
      deep: true,
      handler() {
        // 自动保存隐藏的主题配置
        if (typeof this.saveHiddenStyles === 'function') {
          this.saveHiddenStyles();
        }
      }
    },
    styleOrder: {
      deep: true,
      handler() {
        // 自动保存主题顺序配置
        if (typeof this.saveStyleOrder === 'function') {
          this.saveStyleOrder();
        }
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

    /**
     * 自动修复文本中的 CJK 空格和标点问题
     */
    fixTextSpaces() {
      if (!this.markdownInput) {
        this.showToast('没有可修复的内容', 'error');
        return;
      }

      // 检查 autocorrect 是否可用（全局对象）
      if (typeof autocorrect === 'undefined' || !autocorrect.format) {
        this.showToast('AutoCorrect 未加载', 'error');
        console.error('autocorrect global object not found');
        return;
      }

      try {
        // 记录原始长度
        const originalLength = this.markdownInput.length;

        // 调用 autocorrect 修复（全局对象）
        const fixed = autocorrect.format(this.markdownInput);

        // 更新内容
        this.markdownInput = fixed;

        // 显示结果
        const changeCount = Math.abs(fixed.length - originalLength);
        if (changeCount > 0) {
          this.showToast(`已修复 ${changeCount} 处格式问题`, 'success');
        } else {
          this.showToast('没有发现需要修复的问题', 'success');
        }

        console.log(`AutoCorrect: ${originalLength} -> ${fixed.length} chars (${changeCount} changes)`);
      } catch (err) {
        console.error('AutoCorrect error:', err);
        this.showToast('修复失败: ' + err.message, 'error');
      }
    },

    bindEditorScrollSync() {
      const editor = this.$refs.markdownInput;
      const preview = this.$refs.previewContent;

      if (!editor || !preview || this._scrollSyncHandlers) {
        return;
      }

      this._scrollSyncHandlers = {
        editor: () => {
          this.showLinkedScrollbars(preview);
          this.syncScrollPosition(editor, preview);
        },
        preview: () => {
          this.showLinkedScrollbars(preview);
          this.syncScrollPosition(preview, editor);
        }
      };
      editor.addEventListener('scroll', this._scrollSyncHandlers.editor, { passive: true });
      preview.addEventListener('scroll', this._scrollSyncHandlers.preview, { passive: true });
      this.syncPreviewScroll();
    },

    unbindEditorScrollSync() {
      const editor = this.$refs.markdownInput;
      const preview = this.$refs.previewContent;

      if (this._scrollSyncHandlers) {
        if (editor) {
          editor.removeEventListener('scroll', this._scrollSyncHandlers.editor);
        }
        if (preview) {
          preview.removeEventListener('scroll', this._scrollSyncHandlers.preview);
        }
      }
      this._scrollSyncHandlers = null;
      this.hideLinkedScrollbars();
    },

    syncPreviewScroll() {
      const editor = this.$refs.markdownInput || this.$el?.querySelector('.markdown-input');
      const preview = this.$refs.previewContent || this.$el?.querySelector('.preview-content');

      this.syncScrollPosition(editor, preview);
    },

    syncScrollPosition(source, target) {
      if (!source || !target || this._isSyncingScroll) {
        return;
      }

      const sourceScrollableHeight = source.scrollHeight - source.clientHeight;
      const targetScrollableHeight = target.scrollHeight - target.clientHeight;
      const targetScrollTop = sourceScrollableHeight > 0 && targetScrollableHeight > 0
        ? (source.scrollTop / sourceScrollableHeight) * targetScrollableHeight
        : 0;

      if (Math.abs(target.scrollTop - targetScrollTop) < 1) {
        return;
      }

      this._isSyncingScroll = true;
      target.scrollTop = targetScrollTop;
      window.requestAnimationFrame(() => {
        this._isSyncingScroll = false;
      });
    },

    showLinkedScrollbars(...elements) {
      const scrollElements = elements.filter(Boolean);
      if (!scrollElements.length) {
        return;
      }

      scrollElements.forEach((element) => {
        element.classList.add('is-scrolling');
      });

      if (this._scrollbarHideTimeout) {
        window.clearTimeout(this._scrollbarHideTimeout);
      }

      this._scrollbarHideTimeout = window.setTimeout(() => {
        scrollElements.forEach((element) => {
          element.classList.remove('is-scrolling');
        });
        this._scrollbarHideTimeout = null;
      }, 700);
    },

    hideLinkedScrollbars() {
      if (this._scrollbarHideTimeout) {
        window.clearTimeout(this._scrollbarHideTimeout);
        this._scrollbarHideTimeout = null;
      }

      const preview = this.$refs.previewContent;
      [preview].filter(Boolean).forEach((element) => {
        element.classList.remove('is-scrolling');
      });
    },

    ...SafeEditorMethods
  }
});

editorApp.mount('#app');
