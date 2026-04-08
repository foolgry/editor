/**
 * 分享页与主站共享的渲染工具
 */
(() => {
  const wechatEditorModules = window.WechatEditorModules || {};
  const globalStyles = typeof STYLES !== 'undefined' ? STYLES : {};
  const styleRegistry = wechatEditorModules.STYLES || globalStyles;
  const FALLBACK_MERMAID_LANGUAGES = [
    'mermaid',
    'flowchart',
    'graph',
    'sequenceDiagram',
    'gantt',
    'classDiagram',
    'stateDiagram',
    'erDiagram',
    'journey',
    'pie',
    'gitGraph',
    'requirementDiagram'
  ];

  function getRenderCore() {
    return window.WXMDRenderCore && typeof window.WXMDRenderCore === 'object'
      ? window.WXMDRenderCore
      : null;
  }

  function getMermaidLanguages() {
    const renderCore = getRenderCore();
    if (renderCore && Array.isArray(renderCore.MERMAID_LANGUAGES)) {
      return new Set(renderCore.MERMAID_LANGUAGES);
    }
    return new Set(FALLBACK_MERMAID_LANGUAGES);
  }

  function normalizeStyleAlias(rawValue) {
    return String(rawValue || '')
      .trim()
      .toLowerCase()
      .replace(/[()]/g, '')
      .replace(/[（）]/g, '')
      .replace(/隐藏/g, '')
      .replace(/\s+/g, '-')
      .replace(/_+/g, '-')
      .replace(/-+/g, '-');
  }

  function createStyleAliasMap() {
    const aliases = {
      'claude': 'wechat-anthropic',
      'anthropic': 'wechat-anthropic',
      'claude-song': 'wechat-claude-song',
      'claude-song-serif': 'wechat-claude-song',
      'default': 'wechat-default'
    };

    Object.entries(styleRegistry).forEach(([styleKey, styleConfig]) => {
      const normalizedKey = normalizeStyleAlias(styleKey);
      if (normalizedKey) {
        aliases[normalizedKey] = styleKey;
      }

      const styleName = normalizeStyleAlias(styleConfig && styleConfig.name);
      if (styleName && !aliases[styleName]) {
        aliases[styleName] = styleKey;
      }
    });

    return aliases;
  }

  const styleAliasMap = createStyleAliasMap();

  function resolveStyleKey(styleKey) {
    if (styleKey && styleRegistry[styleKey]) {
      return styleKey;
    }

    const normalizedStyleKey = normalizeStyleAlias(styleKey);
    const resolvedStyleKey = styleAliasMap[normalizedStyleKey];

    if (resolvedStyleKey && styleRegistry[resolvedStyleKey]) {
      return resolvedStyleKey;
    }

    return styleRegistry['wechat-default'] ? 'wechat-default' : Object.keys(styleRegistry)[0];
  }

  function createMarkdownRenderer(options = {}) {
    const renderCore = getRenderCore();
    if (renderCore && typeof renderCore.createMarkdownParser === 'function') {
      return renderCore.createMarkdownParser({
        markdownit: options.markdownit || window.markdownit,
        hljs: options.hljs || (typeof hljs !== 'undefined' ? hljs : null)
      });
    }

    const escapeHtml = window.markdownit().utils.escapeHtml;
    const mermaidLanguages = getMermaidLanguages();

    return window.markdownit({
      html: true,
      linkify: true,
      typographer: false,
      highlight: function(str, lang) {
        if (lang && mermaidLanguages.has(lang)) {
          const escapedSource = escapeHtml(str);
          return `<div class="mermaid" style="background: #fff; padding: 20px; border-radius: 8px; margin: 20px 0; text-align: center; overflow-x: auto;">${escapedSource}</div>`;
        }

        const dots = '<div style="display: flex; align-items: center; gap: 6px; padding: 10px 12px; background: #2a2c33; border-bottom: 1px solid #1e1f24;"><span style="width: 12px; height: 12px; border-radius: 50%; background: #ff5f56;"></span><span style="width: 12px; height: 12px; border-radius: 50%; background: #ffbd2e;"></span><span style="width: 12px; height: 12px; border-radius: 50%; background: #27c93f;"></span></div>';

        let codeContent = '';
        if (lang && typeof hljs !== 'undefined') {
          try {
            if (hljs.getLanguage(lang)) {
              codeContent = hljs.highlight(str, { language: lang }).value;
            } else {
              codeContent = escapeHtml(str);
            }
          } catch (error) {
            codeContent = escapeHtml(str);
          }
        } else {
          codeContent = escapeHtml(str);
        }

        return `<div style="margin: 20px 0; border-radius: 8px; overflow: hidden; background: #383a42; box-shadow: 0 2px 8px rgba(0,0,0,0.15);">${dots}<div style="padding: 16px; overflow-x: auto; background: #383a42;"><code style="display: block; color: #abb2bf; font-family: 'SF Mono', Monaco, 'Cascadia Code', Consolas, monospace; font-size: 14px; line-height: 1.6; white-space: pre;">${codeContent}</code></div></div>`;
      }
    });
  }

  function patchMarkdownScanner(md) {
    const renderCore = getRenderCore();
    if (renderCore && typeof renderCore.patchMarkdownScanner === 'function') {
      renderCore.patchMarkdownScanner(md);
    }
  }

  function preprocessMarkdown(content) {
    const renderCore = getRenderCore();
    if (renderCore && typeof renderCore.preprocessMarkdown === 'function') {
      return renderCore.preprocessMarkdown(content);
    }
    return String(content || '');
  }

  function stripCitationMarkers(content) {
    const renderCore = getRenderCore();
    if (renderCore && typeof renderCore.stripCitationMarkers === 'function') {
      return renderCore.stripCitationMarkers(content);
    }
    return String(content || '');
  }

  function applyInlineStyles(html, options = {}) {
    const renderCore = getRenderCore();
    if (renderCore && typeof renderCore.applyInlineStyles === 'function') {
      return renderCore.applyInlineStyles(html, {
        styles: options.styles || styleRegistry,
        styleKey: resolveStyleKey(options.styleKey),
        parseHtml: options.parseHtml
      });
    }
    return html;
  }

  function renderMarkdown(markdown, options = {}) {
    const renderCore = getRenderCore();
    if (renderCore && typeof renderCore.renderMarkdown === 'function') {
      return renderCore.renderMarkdown(markdown, {
        styles: options.styles || styleRegistry,
        styleKey: resolveStyleKey(options.styleKey),
        markdownit: options.markdownit || window.markdownit,
        hljs: options.hljs || (typeof hljs !== 'undefined' ? hljs : null),
        parseHtml: options.parseHtml
      });
    }

    const md = options.md || createMarkdownRenderer(options);
    return applyInlineStyles(md.render(preprocessMarkdown(markdown)), options);
  }

  window.WechatEditorModules = window.WechatEditorModules || {};
  window.WechatEditorModules.RenderUtils = {
    MERMAID_LANGUAGES: getMermaidLanguages(),
    createMarkdownRenderer,
    patchMarkdownScanner,
    preprocessMarkdown,
    stripCitationMarkers,
    applyInlineStyles,
    renderMarkdown,
    resolveStyleKey,
    normalizeStyleAlias
  };
})();
