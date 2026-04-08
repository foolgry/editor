/**
 * 分享页入口
 */
(() => {
  const { createApp } = Vue;
  const wechatEditorModules = window.WechatEditorModules || {};
  const RenderUtils = wechatEditorModules.RenderUtils || {};
  const EditorMethods = wechatEditorModules.EditorMethods || {};
  const sharedPayload = window.__WX_EDITOR_SHARE__ || {};
  const createMarkdownRenderer = RenderUtils.createMarkdownRenderer || function() {
    throw new Error('RenderUtils 未加载');
  };
  const resolveStyleKey = RenderUtils.resolveStyleKey || function(styleKey) {
    return styleKey || 'wechat-default';
  };

  createApp({
    data() {
      return {
        loading: true,
        error: null,
        markdownInput: typeof sharedPayload.content === 'string' ? sharedPayload.content : '',
        renderedContent: '',
        currentStyle: resolveStyleKey(sharedPayload.style),
        md: null,
        imageStore: null,
        imageIdToObjectURL: {},
        mermaidInitialized: false,
        articleHistory: [],
        currentArticleId: null
      };
    },

    async mounted() {
      try {
        this.currentStyle = resolveStyleKey(this.currentStyle);
        this.md = createMarkdownRenderer();
        if (typeof this.patchMarkdownScanner === 'function') {
          this.patchMarkdownScanner(this.md);
        }
        await this.renderMarkdown();
      } catch (error) {
        console.error('分享页渲染失败:', error);
        this.error = '内容渲染失败';
      } finally {
        this.loading = false;
      }
    },

    methods: {
      ...EditorMethods,
      saveToHistory() {},
      showToast(message, type = 'success') {
        const logger = type === 'error' ? console.error : console.log;
        logger(message);
      }
    }
  }).mount('#app');
})();
