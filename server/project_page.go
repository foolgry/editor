package main

import (
	"encoding/json"
	"html"
	"strings"
)

// projectPageMeta 项目页元数据，整体编码为 JSON 字符串嵌入页面（__WX_PROJECT_JSON__），
// 供前端 JS 读取项目 id/名称与当前篇 id
type projectPageMeta struct {
	ProjectID      string `json:"projectId"`
	ProjectName    string `json:"projectName"`
	CurrentShareID string `json:"currentShareId"`
}

// buildProjectTOCHTML 服务端生成目录 HTML：每项两行（标题 + 日期），当前篇高亮（current 类），
// 链接为真实深链（无 JS 也可直接跳转）。所有动态值经 html.EscapeString 转义。
// shares 顺序即创建时间倒序（listProjectShares 已排序），此处直接使用。
func buildProjectTOCHTML(project Project, shares []ShareListItem, currentShare *Share) string {
	if len(shares) == 0 {
		return `<div class="toc-empty">暂无文章</div>`
	}

	currentID := ""
	if currentShare != nil {
		currentID = currentShare.ID
	}

	var b strings.Builder
	for _, s := range shares {
		class := "toc-item"
		if s.ID == currentID {
			class += " current"
		}
		b.WriteString(`<a class="` + class + `" href="/p/` + html.EscapeString(project.ID) + `/` + html.EscapeString(s.ID) + `" data-sid="` + html.EscapeString(s.ID) + `">`)
		b.WriteString(`<span class="toc-title">` + html.EscapeString(s.Title) + `</span>`)
		b.WriteString(`<span class="toc-date">` + s.CreatedAt.Format("2006-01-02") + `</span>`)
		b.WriteString(`</a>`)
	}
	return b.String()
}

// projectArticleTemplate 文章区 Vue 模板（与单篇分享页同一结构）：
// loading / error / 渲染结果三分支，挂载后由 Vue 接管
const projectArticleTemplate = `<div v-if="loading" class="loading">
        <div class="spinner"></div>
        <p>正在加载内容...</p>
      </div>

      <div v-else-if="error" class="error">
        <div class="error-icon">😕</div>
        <h3>{{ error }}</h3>
      </div>

      <div v-else class="wxmd-article" v-html="renderedContent"></div>`

// projectPageTemplate 项目聚合页模板：左侧目录 + 右侧文章区 + 移动端抽屉。
// 资源加载与单篇分享页（generateSharePageHTML）完全一致。
const projectPageTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>__WX_PAGE_TITLE__</title>
  <meta name="description" content="__WX_PROJECT_NAME__">
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  <link rel="alternate icon" href="/favicon.svg">
  <link rel="mask-icon" href="/favicon.svg" color="#0066FF">

  <!-- 代码高亮样式（自托管） -->
  <link rel="stylesheet" href="/lib/vendor/atom-one-dark.min.css">

  <!-- 核心库（自托管 + defer 按序执行，不阻塞首屏渲染） -->
  <script src="/lib/vendor/markdown-it.min.js" defer></script>
  <script src="/lib/vendor/highlight.min.js" defer></script>
  <script src="/lib/vendor/vue.global.prod.js" defer></script>

  <!-- mermaid（3MB+）按需加载，仅当文章含图表时才引入（见下方 renderMermaid） -->
  <script src="/lib/lazy-loader.js" defer></script>

  <style>
    :root {
      --color-primary: #111827;
      --color-secondary: #6b7280;
      --color-accent: #2563eb;
      --color-bg: #FAFAFA;
      --color-surface: #FFF;
      --color-border: #E0E0E0;
      --font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", "PingFang SC", "Microsoft YaHei", sans-serif;
      --font-mono: "SF Mono", Monaco, "Cascadia Code", "Consolas", monospace;
    }

    * {
      margin: 0;
      padding: 0;
      box-sizing: border-box;
    }

    body {
      font-family: var(--font-sans);
      font-size: 15px;
      line-height: 1.6;
      color: var(--color-primary);
      background-color: var(--color-bg);
      -webkit-font-smoothing: antialiased;
    }

    /* ===== 左侧目录栏（中性样式，不套用文章 STYLES，不受文章 style 影响） ===== */
    .sidebar {
      position: fixed;
      top: 0;
      left: 0;
      bottom: 0;
      width: 280px;
      background: var(--color-surface);
      border-right: 1px solid var(--color-border);
      overflow-y: auto;
      z-index: 120;
    }

    .toc-header {
      position: sticky;
      top: 0;
      background: var(--color-surface);
      padding: 16px 20px;
      font-size: 16px;
      font-weight: 700;
      border-bottom: 1px solid var(--color-border);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .toc {
      padding: 8px 0 24px;
    }

    .toc-item {
      display: block;
      padding: 10px 20px 10px 17px;
      text-decoration: none;
      border-left: 3px solid transparent;
    }

    .toc-item:hover {
      background: #f9fafb;
    }

    .toc-item.current {
      border-left-color: var(--color-accent);
      background: #f0f7ff;
    }

    .toc-title {
      display: block;
      font-size: 14px;
      color: var(--color-primary);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .toc-item.current .toc-title {
      font-weight: 600;
    }

    .toc-date {
      display: block;
      margin-top: 2px;
      font-size: 12px;
      color: var(--color-secondary);
    }

    .toc-empty {
      padding: 16px 20px;
      font-size: 13px;
      color: var(--color-secondary);
    }

    /* ===== 右侧文章区 ===== */
    .main-area {
      margin-left: 280px;
    }

    .content {
      max-width: 800px;
      margin: 0 auto;
      padding: 40px 24px 80px;
    }

    /* 移动端顶栏（桌面隐藏，仅手机显示） */
    .mobile-bar {
      display: none;
    }

    /* 抽屉半透明遮罩（默认隐藏，仅手机抽屉展开时显示） */
    .overlay {
      display: none;
    }

    /* 切换文章失败时的行内提示 */
    .load-error {
      display: none;
      position: fixed;
      top: 16px;
      left: 50%;
      transform: translateX(-50%);
      z-index: 300;
      max-width: 90vw;
      padding: 8px 16px;
      background: #fef2f2;
      color: #b91c1c;
      border: 1px solid #fecaca;
      border-radius: 8px;
      font-size: 13px;
    }

    /* 空项目占位 */
    .empty-state {
      min-height: 60vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      color: var(--color-secondary);
    }

    .empty-icon {
      font-size: 48px;
      margin-bottom: 16px;
    }

    /* ===== 以下为文章渲染相关样式（与单篇分享页一致） ===== */
    .loading {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 400px;
      color: var(--color-secondary);
    }

    .spinner {
      width: 40px;
      height: 40px;
      border: 3px solid var(--color-border);
      border-top-color: var(--color-accent);
      border-radius: 50%;
      animation: spin 1s linear infinite;
      margin-bottom: 16px;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    .error {
      text-align: center;
      padding: 60px 24px;
      color: var(--color-secondary);
    }

    .error-icon {
      font-size: 48px;
      margin-bottom: 16px;
    }

    /* Mermaid 图表样式 */
    .mermaid {
      background: #fff !important;
      border-radius: 8px;
      margin: 20px 0;
      padding: 20px;
      text-align: center;
      overflow-x: auto;
    }

    .mermaid svg {
      max-width: 100%;
      height: auto;
      display: inline-block;
    }

    /* ===== 手机断点：目录收进抽屉，第一眼看到文章内容本身 ===== */
    @media (max-width: 768px) {
      .mobile-bar {
        display: flex;
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        height: 48px;
        align-items: center;
        gap: 12px;
        padding: 0 12px;
        background: var(--color-surface);
        border-bottom: 1px solid var(--color-border);
        z-index: 150;
      }

      .menu-btn {
        width: 36px;
        height: 36px;
        border: none;
        background: transparent;
        font-size: 20px;
        line-height: 1;
        color: var(--color-primary);
        cursor: pointer;
      }

      .mobile-title {
        flex: 1;
        font-size: 15px;
        font-weight: 600;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .sidebar {
        transform: translateX(-100%);
        transition: transform 0.25s ease;
        z-index: 200;
      }

      body.drawer-open .sidebar {
        transform: translateX(0);
        box-shadow: 0 0 24px rgba(17, 24, 39, 0.12);
      }

      .overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(17, 24, 39, 0.45);
        z-index: 180;
      }

      body.drawer-open .overlay {
        display: block;
      }

      body.drawer-open {
        overflow: hidden;
      }

      .main-area {
        margin-left: 0;
        padding-top: 48px;
      }

      .content {
        padding: 24px 16px 64px;
      }

      .load-error {
        top: 60px;
      }
    }
  </style>
</head>
<body>
  <!-- 左侧目录栏（手机端为抽屉） -->
  <aside class="sidebar" id="sidebar">
    <div class="toc-header">__WX_PROJECT_NAME__</div>
    <nav class="toc" id="toc">
      __WX_TOC_ITEMS__
    </nav>
  </aside>

  <!-- 抽屉遮罩（仅手机） -->
  <div class="overlay" id="overlay"></div>

  <!-- 手机顶栏：项目名 + 菜单按钮 -->
  <header class="mobile-bar">
    <button class="menu-btn" id="menuBtn" aria-label="打开目录">☰</button>
    <span class="mobile-title">__WX_PROJECT_NAME__</span>
  </header>

  <!-- 右侧文章区 -->
  <main class="main-area">
    <div class="load-error" id="loadError">加载失败，请重试</div>
    <article class="content" id="app">
      __WX_APP_INNER__
    </article>
  </main>

  <script src="/lib/lazy-loader.js" defer></script>
  <script src="/render-core.js" defer></script>
  <script src="/styles.js" defer></script>
  <script src="/outline.js" defer></script>
  <script>
    // 库脚本均带 defer，会在 DOMContentLoaded 之前按文档顺序执行完毕，
    // 因此在 DOMContentLoaded 回调里启动时 Vue / markdown-it / hljs / STYLES 已就绪
    document.addEventListener('DOMContentLoaded', function() {
    var PROJECT = JSON.parse(__WX_PROJECT_JSON__);

    var tocEl = document.getElementById('toc');
    var menuBtn = document.getElementById('menuBtn');
    var overlayEl = document.getElementById('overlay');
    var loadErrorEl = document.getElementById('loadError');
    var loadErrorTimer = null;

    // ===== 移动端抽屉 =====
    function openDrawer() {
      document.body.classList.add('drawer-open');
    }

    function closeDrawer() {
      document.body.classList.remove('drawer-open');
    }

    if (menuBtn) {
      menuBtn.addEventListener('click', openDrawer);
    }
    if (overlayEl) {
      overlayEl.addEventListener('click', closeDrawer);
    }

    // ===== 引用标记净化（与 Go 侧 stripCitationMarkers 同一规则，用于 fetch 回来的内容） =====
    function stripCitationMarkers(content) {
      return String(content || '').replace(/\uE200cite\uE202[^\uE201]*\uE201/g, '');
    }

    // ===== 目录条目查找（遍历比较 data-sid，不拼接选择器） =====
    function findItem(sid) {
      if (!tocEl) return null;
      var links = tocEl.querySelectorAll('a.toc-item');
      for (var i = 0; i < links.length; i++) {
        if (links[i].getAttribute('data-sid') === sid) return links[i];
      }
      return null;
    }

    // 当前篇标题（取目录条目文本，与服务端渲染一致），找不到时退回项目名
    function tocTitle(sid) {
      var item = findItem(sid);
      if (item) {
        var titleEl = item.querySelector('.toc-title');
        if (titleEl) return titleEl.textContent;
      }
      return PROJECT.projectName;
    }

    function firstSid() {
      var first = tocEl ? tocEl.querySelector('a.toc-item') : null;
      return first ? first.getAttribute('data-sid') : null;
    }

    // 更新目录高亮
    function setActive(sid) {
      if (!tocEl) return;
      var links = tocEl.querySelectorAll('a.toc-item');
      for (var i = 0; i < links.length; i++) {
        if (links[i].getAttribute('data-sid') === sid) {
          links[i].classList.add('current');
        } else {
          links[i].classList.remove('current');
        }
      }
    }

    // 切换失败的行内提示（保持当前篇不回退）
    function showLoadError(show) {
      if (!loadErrorEl) return;
      loadErrorEl.style.display = show ? 'block' : 'none';
      if (loadErrorTimer) {
        clearTimeout(loadErrorTimer);
        loadErrorTimer = null;
      }
      if (show) {
        loadErrorTimer = setTimeout(function() {
          loadErrorEl.style.display = 'none';
        }, 3000);
      }
    }

    // ===== 文章渲染器（与单篇分享页同一管线），仅在项目内有文章时挂载 =====
    var vm = null;
    if (PROJECT.currentShareId) {
      var app = Vue.createApp({
        data() {
          return {
            loading: true,
            error: null,
            renderedContent: '',
            markdownContent: __WX_EDITOR_MARKDOWN_CONTENT__,
            style: __WX_EDITOR_STYLE__,
            md: null
          };
        },

        mounted() {
          this.initMarkdown();
          this.renderContent();
        },

        methods: {
          initMarkdown() {
            const renderCore = window.WXMDRenderCore;
            if (renderCore && typeof renderCore.createMarkdownParser === 'function') {
              this.md = renderCore.createMarkdownParser({
                markdownit: window.markdownit,
                hljs: typeof hljs !== 'undefined' ? hljs : null
              });
              return;
            }

            this.md = window.markdownit({
              html: true,
              linkify: true,
              typographer: false
            });
          },

          async renderContent() {
            try {
              const renderCore = window.WXMDRenderCore;
              let html = '';
              if (renderCore && typeof renderCore.renderMarkdown === 'function') {
                html = renderCore.renderMarkdown(this.markdownContent, {
                  md: this.md,
                  styles: STYLES,
                  styleKey: this.style
                });
              } else {
                const processedContent = renderCore && typeof renderCore.preprocessMarkdown === 'function'
                  ? renderCore.preprocessMarkdown(this.markdownContent)
                  : this.markdownContent;
                html = this.applyInlineStyles(this.md.render(processedContent));
              }
              this.renderedContent = html;
              this.loading = false;

              // 等待 DOM 更新后渲染 Mermaid 图表
              await this.$nextTick();
              this.renderMermaid();
            } catch (err) {
              console.error('渲染失败:', err);
              this.error = '内容渲染失败';
              this.loading = false;
            }
          },

          async renderMermaid() {
            // 页面中没有 mermaid 图表时不加载 mermaid 库（3MB+）
            if (!document.querySelector('.mermaid')) {
              return;
            }

            try {
              // mermaid 为按需加载，首屏不引入
              let mermaidLib = typeof mermaid !== 'undefined' ? mermaid : null;
              if (!mermaidLib && window.WXMDLazy) {
                mermaidLib = await window.WXMDLazy.loadMermaid();
              }
              if (!mermaidLib) {
                return;
              }

              mermaidLib.initialize({
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

              // 查找所有未渲染的 mermaid 图表
              const mermaidElements = document.querySelectorAll('.mermaid:not([data-processed])');
              if (mermaidElements.length > 0) {
                mermaidLib.run({
                  querySelector: '.mermaid'
                });
              }
            } catch (err) {
              console.error('Mermaid 渲染失败:', err);
            }
          },

          applyInlineStyles(html) {
            const renderCore = window.WXMDRenderCore;
            if (renderCore && typeof renderCore.applyInlineStyles === 'function') {
              return renderCore.applyInlineStyles(html, {
                styles: STYLES,
                styleKey: this.style
              });
            }

            const style = STYLES[this.style] ? STYLES[this.style].styles : STYLES['wechat-default'].styles;
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, 'text/html');

            Object.keys(style).forEach(selector => {
              if (selector === 'pre' || selector === 'code' || selector === 'pre code') {
                return;
              }

              const elements = doc.querySelectorAll(selector);
              elements.forEach(el => {
                const currentStyle = el.getAttribute('style') || '';
                el.setAttribute('style', currentStyle + '; ' + style[selector]);
              });
            });

            // 标题内的行内元素统一继承标题颜色
            const headings = doc.querySelectorAll('h1, h2, h3, h4, h5, h6');
            const headingInlineOverrides = {
              strong: 'font-weight: 700; color: inherit !important; background-color: transparent !important;',
              em: 'font-style: italic; color: inherit !important; background-color: transparent !important;',
              a: 'color: inherit !important; text-decoration: none !important; border-bottom: 1px solid currentColor !important; background-color: transparent !important;',
              code: 'color: inherit !important; background-color: transparent !important; border: none !important; padding: 0 !important;',
              span: 'color: inherit !important; background-color: transparent !important;',
              b: 'font-weight: 700; color: inherit !important; background-color: transparent !important;',
              i: 'font-style: italic; color: inherit !important; background-color: transparent !important;',
            };
            const headingInlineSelectorList = Object.keys(headingInlineOverrides).join(', ');

            headings.forEach(heading => {
              const inlineNodes = heading.querySelectorAll(headingInlineSelectorList);
              inlineNodes.forEach(node => {
                const tag = node.tagName.toLowerCase();
                let override = headingInlineOverrides[tag];
                if (!override) return;

                const currentStyle = node.getAttribute('style') || '';
                const sanitizedStyle = currentStyle
                  .replace(/color:\s*[^;]+;?/gi, '')
                  .replace(/background(?:-color)?:\s*[^;]+;?/gi, '')
                  .replace(/border(?:-bottom)?:\s*[^;]+;?/gi, '')
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

          // 切换文章：注入新内容与样式并重新渲染
          async setContent(content, style) {
            this.markdownContent = content;
            this.style = style;
            this.error = null;
            this.loading = true;
            await this.renderContent();
          }
        }
      });
      vm = app.mount('#app');
    }

    // ===== 切换文章：fetch 公开接口 → 重渲染 → pushState → 更新高亮 → 滚顶/收抽屉 =====
    function switchShare(sid, push) {
      if (!vm || !sid || sid === PROJECT.currentShareId) {
        // 未挂载渲染器 / 同篇点击：仅收起抽屉并回到顶部，不重复请求与 pushState
        closeDrawer();
        window.scrollTo(0, 0);
        return;
      }

      showLoadError(false);
      vm.loading = true;

      fetch('/api/share/' + encodeURIComponent(sid))
        .then(function(res) {
          if (!res.ok) throw new Error('HTTP ' + res.status);
          return res.json();
        })
        .then(function(data) {
          PROJECT.currentShareId = sid;
          setActive(sid);
          document.title = tocTitle(sid) + ' · ' + PROJECT.projectName;
          vm.setContent(stripCitationMarkers(data.content), String(data.style || ''));
          if (push) {
            history.pushState(null, '', '/p/' + PROJECT.projectId + '/' + sid);
          }
          closeDrawer();
          window.scrollTo(0, 0);
        })
        .catch(function(err) {
          console.error('切换文章失败:', err);
          vm.loading = false;
          showLoadError(true); // 提示失败，保持当前篇
        });
    }

    // 目录点击：有 JS 时拦截走无刷新切换，无 JS 时退化为真实链接跳转
    if (tocEl) {
      tocEl.addEventListener('click', function(event) {
        var link = event.target && event.target.closest ? event.target.closest('a.toc-item') : null;
        if (!link || !tocEl.contains(link)) return;
        event.preventDefault();
        switchShare(link.getAttribute('data-sid'), true);
      });
    }

    // 浏览器前进/后退：按地址栏解析目标篇，等价执行一次切换（不再 pushState）
    window.addEventListener('popstate', function() {
      var match = location.pathname.match(/^\/p\/([^/]+)(?:\/([^/]+))?\/?$/);
      if (!match || match[1] !== PROJECT.projectId) return;
      var sid = match[2] || firstSid();
      if (sid) {
        switchShare(sid, false);
      }
    });

    // 右侧大纲（从渲染后的文章里取 h2/h3/h4，默认展开、可收起）；切换文章后自动重建
    if (window.WXMDOutline) window.WXMDOutline.mount();
    });
  </script>
</body>
</html>`

// generateProjectPageHTML 生成项目聚合页面 HTML：
// 桌面为左侧固定 280px 目录 + 右侧 max-width 800px 文章区；手机目录收进抽屉。
// 目录由服务端渲染（标题 + 日期，倒序，当前篇高亮），文章渲染复用单篇分享页的
// 渲染管线（markdown-it / render-core / styles + mermaid 按需加载）。
func generateProjectPageHTML(project Project, shares []ShareListItem, currentShare *Share) string {
	tocHTML := buildProjectTOCHTML(project, shares, currentShare)

	// 当前篇初始内容（净化后嵌入，与分享页一致）；空项目嵌入空串且不挂渲染器
	cleanContent := ""
	style := ""
	// 文章区初始结构：有文章时为 Vue 模板（挂载后接管），空项目时为服务端空态占位
	appInnerHTML := `<div class="empty-state"><div class="empty-icon">📄</div><p>该项目暂无文章</p></div>`
	if currentShare != nil {
		cleanContent = stripCitationMarkers(currentShare.Content)
		style = currentShare.Style
		appInnerHTML = projectArticleTemplate
	}

	// 项目元数据整体编码为 JSON 字符串后经 inlineJSONString 嵌入
	meta := projectPageMeta{ProjectID: project.ID, ProjectName: project.Name}
	if currentShare != nil {
		meta.CurrentShareID = currentShare.ID
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		metaJSON = []byte("{}")
	}

	// 页面标题：有当前篇时为"标题 · 项目名"，空项目为项目名（与切换文章后的行为一致）
	pageTitle := project.Name
	if currentShare != nil {
		title := extractTitleFromMarkdown(stripCitationMarkers(currentShare.Content))
		if title == "" {
			title = "分享的文章"
		}
		pageTitle = title + " · " + project.Name
	}

	replacer := strings.NewReplacer(
		"__WX_PAGE_TITLE__", html.EscapeString(pageTitle),
		"__WX_PROJECT_NAME__", html.EscapeString(project.Name),
		"__WX_PROJECT_JSON__", inlineJSONString(string(metaJSON)),
		"__WX_TOC_ITEMS__", tocHTML,
		"__WX_APP_INNER__", appInnerHTML,
		"__WX_EDITOR_MARKDOWN_CONTENT__", inlineJSONString(cleanContent),
		"__WX_EDITOR_STYLE__", inlineJSONString(style),
	)

	return replacer.Replace(projectPageTemplate)
}
