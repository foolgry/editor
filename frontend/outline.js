/**
 * 阅读页右侧大纲
 *
 * 从渲染完成的文章里取 h2/h3/h4 生成右侧大纲：默认展开，可收起（选择记在 localStorage）。
 * 分享页（/s/:id）与项目页（/p/:id/:sid）共用同一份实现，项目页切换文章后由
 * MutationObserver 自动重建，页面自身不需要知道大纲的存在。
 *
 * 窄屏（正文右侧放不下面板）时整个大纲不出现，避免压住正文；手机上项目页本就有抽屉目录。
 */
(function () {
  'use strict';

  var STORAGE_KEY = 'wxmd-outline-collapsed';
  var STYLE_ID = 'wxmd-outline-style';
  var HEADING_SELECTOR = 'h2, h3, h4';

  // 只有一个标题时大纲没有导航价值，不出
  var MIN_HEADINGS = 2;

  // 页面顶部可能存在的粘性栏，用于计算滚动落点
  var STICKY_SELECTOR = '.header, .mobile-bar';

  // 文章切换时 DOM 会在短时间内连续变动，攒一下再重建
  var REBUILD_DELAY = 150;
  var RESIZE_DELAY = 120;

  // 以下尺寸与 CSS 保持一致：既用于判断右侧是否放得下，也用于算出面板贴正文右缘的落点
  var PANEL_WIDTH = 200;
  var PANEL_GAP = 20;
  var TAB_WIDTH = 72;
  var TAB_GAP = 12;

  var CSS = `
.wxmd-outline,
.wxmd-outline-tab {
  --wxmd-ol-accent: var(--color-accent, #2563eb);
  --wxmd-ol-text: var(--color-primary, #111827);
  --wxmd-ol-muted: var(--color-secondary, #6b7280);
  --wxmd-ol-surface: var(--color-surface, #fff);
  --wxmd-ol-border: var(--color-border, #e0e0e0);
  --wxmd-ol-font: var(--font-sans, -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", "PingFang SC", "Microsoft YaHei", sans-serif);
  box-sizing: border-box;
  font-family: var(--wxmd-ol-font);
}

.wxmd-outline[hidden],
.wxmd-outline-tab[hidden] {
  display: none !important;
}

/* ===== 展开态：贴在正文右缘的面板 ===== */
.wxmd-outline {
  position: fixed;
  top: 50%;
  transform: translateY(-50%);
  z-index: 90;
  display: flex;
  flex-direction: column;
  width: 200px;
  max-height: calc(100vh - 180px);
  background: var(--wxmd-ol-surface);
  border: 1px solid var(--wxmd-ol-border);
  border-radius: 10px;
  box-shadow: 0 1px 2px rgba(17, 24, 39, 0.04), 0 8px 24px rgba(17, 24, 39, 0.07);
  overflow: hidden;
}

.wxmd-outline-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 9px 8px 9px 16px;
  border-bottom: 1px solid var(--wxmd-ol-border);
}

.wxmd-outline-title {
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: var(--wxmd-ol-text);
}

.wxmd-outline-toggle {
  flex: none;
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--wxmd-ol-muted);
  font-family: inherit;
  font-size: 16px;
  line-height: 1;
  cursor: pointer;
}

.wxmd-outline-toggle:hover {
  background: rgba(17, 24, 39, 0.06);
  color: var(--wxmd-ol-text);
}

.wxmd-outline-body {
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 6px 0 8px;
}

.wxmd-outline-item {
  display: block;
  width: 100%;
  padding: 5px 14px;
  border: none;
  border-left: 2px solid transparent;
  background: transparent;
  color: var(--wxmd-ol-muted);
  font-family: inherit;
  font-size: 13px;
  line-height: 1.5;
  text-align: left;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wxmd-outline-item:hover {
  background: rgba(17, 24, 39, 0.04);
  color: var(--wxmd-ol-text);
}

.wxmd-outline-item.is-active {
  background: rgba(37, 99, 235, 0.07);
  background: color-mix(in srgb, var(--wxmd-ol-accent) 8%, transparent);
  border-left-color: var(--wxmd-ol-accent);
  color: var(--wxmd-ol-accent);
  font-weight: 500;
}

.wxmd-outline-l3 {
  padding-left: 28px;
  font-size: 12.5px;
}

.wxmd-outline-l4 {
  padding-left: 42px;
  font-size: 12px;
}

.wxmd-outline-toggle:focus-visible,
.wxmd-outline-item:focus-visible,
.wxmd-outline-tab:focus-visible {
  outline: 2px solid var(--wxmd-ol-accent);
  outline-offset: -2px;
}

/* ===== 收起态：只留一枚悬浮标签，点开即回到展开态 ===== */
.wxmd-outline-tab {
  position: fixed;
  top: 50%;
  transform: translateY(-50%);
  z-index: 90;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  width: 72px;
  height: 34px;
  padding: 0;
  border: 1px solid var(--wxmd-ol-border);
  border-radius: 17px;
  background: var(--wxmd-ol-surface);
  box-shadow: 0 2px 8px rgba(17, 24, 39, 0.08);
  color: var(--wxmd-ol-muted);
  font-family: inherit;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
}

.wxmd-outline-tab:hover {
  color: var(--wxmd-ol-text);
}
`;

  var instance = null;

  function injectStyle() {
    if (document.getElementById(STYLE_ID)) return;
    var style = document.createElement('style');
    style.id = STYLE_ID;
    style.textContent = CSS;
    document.head.appendChild(style);
  }

  function escapeHTML(value) {
    return String(value).replace(/[&<>"']/g, function (ch) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch];
    });
  }

  function readCollapsed() {
    try {
      return localStorage.getItem(STORAGE_KEY) === '1';
    } catch (err) {
      return false;
    }
  }

  function writeCollapsed(value) {
    try {
      localStorage.setItem(STORAGE_KEY, value ? '1' : '0');
    } catch (err) {
      // 隐私模式下 localStorage 不可用，本次会话内仍然生效
    }
  }

  /**
   * mount 挂载大纲。options 均可省略，默认值同时适用于分享页与项目页：
   *   root    观察主体（文章渲染的宿主），文章切换后据此重建大纲
   *   article 正文容器，用于测量右侧剩余空间（不压住正文的判据）
   *   sticky  页面顶部粘性栏，用于计算滚动落点
   */
  function mount(options) {
    if (instance) return instance;

    var opts = options || {};
    var rootSelector = opts.root || '#app';
    var articleSelector = opts.article || '.wxmd-article';
    var stickySelector = opts.sticky || STICKY_SELECTOR;

    injectStyle();

    var panel = document.createElement('div');
    panel.className = 'wxmd-outline';
    panel.hidden = true;
    panel.innerHTML =
      '<div class="wxmd-outline-head">' +
      '<span class="wxmd-outline-title">大纲</span>' +
      '<button type="button" class="wxmd-outline-toggle" aria-label="收起大纲" aria-expanded="true">›</button>' +
      '</div>' +
      '<nav class="wxmd-outline-body" aria-label="文章大纲"></nav>';

    var tab = document.createElement('button');
    tab.type = 'button';
    tab.className = 'wxmd-outline-tab';
    tab.hidden = true;
    tab.setAttribute('aria-label', '展开大纲');
    tab.textContent = '‹ 大纲';

    document.body.appendChild(panel);
    document.body.appendChild(tab);

    var bodyEl = panel.querySelector('.wxmd-outline-body');
    var toggleEl = panel.querySelector('.wxmd-outline-toggle');

    var headings = [];
    var signature = '';
    var activeIndex = -1;
    var collapsed = readCollapsed();
    var rebuildTimer = null;
    var resizeTimer = null;
    var scrollFrame = null;

    // ===== 采集与渲染 =====

    function collect(host) {
      var nodes = host.querySelectorAll(HEADING_SELECTOR);
      var list = [];
      for (var i = 0; i < nodes.length; i++) {
        var text = (nodes[i].textContent || '').replace(/\s+/g, ' ').trim();
        if (!text) continue;
        list.push({ el: nodes[i], level: Number(nodes[i].tagName.charAt(1)), text: text });
      }
      return list;
    }

    function renderItems() {
      var parts = [];
      for (var i = 0; i < headings.length; i++) {
        var item = headings[i];
        parts.push(
          '<button type="button" class="wxmd-outline-item wxmd-outline-l' + item.level + '"' +
          ' data-index="' + i + '" title="' + escapeHTML(item.text) + '">' +
          escapeHTML(item.text) +
          '</button>'
        );
      }
      bodyEl.innerHTML = parts.join('');
      bodyEl.scrollTop = 0;
      activeIndex = -1;
    }

    // ===== 版式：只有正文右侧放得下面板时才出现 =====

    function articleBox() {
      return document.querySelector(articleSelector);
    }

    function metric() {
      var box = articleBox();
      if (!box) return { room: 0, box: null };
      var width = document.documentElement.clientWidth;
      return { room: width - box.getBoundingClientRect().right, box: box };
    }

    function layout() {
      var result = metric();
      var usable = headings.length >= MIN_HEADINGS && result.box && result.room >= PANEL_GAP + PANEL_WIDTH;

      panel.hidden = !usable || collapsed;
      tab.hidden = !usable || !collapsed;

      if (panel.hidden && tab.hidden) return;

      // 贴着正文右缘落位：正文居中时不会漂到屏幕边上，正文靠边时也不会压到字
      var room = result.room;
      if (!panel.hidden) {
        panel.style.right = Math.max(0, room - PANEL_GAP - PANEL_WIDTH) + 'px';
      } else {
        tab.style.right = Math.max(0, room - TAB_GAP - TAB_WIDTH) + 'px';
      }
    }

    // ===== 滚动高亮 =====

    function stickyHeight() {
      var nodes = document.querySelectorAll(stickySelector);
      var height = 0;
      for (var i = 0; i < nodes.length; i++) {
        if (nodes[i].offsetHeight > height) height = nodes[i].offsetHeight;
      }
      return height;
    }

    function atBottom() {
      var doc = document.documentElement;
      return window.scrollY + window.innerHeight >= doc.scrollHeight - 4;
    }

    function setActive(index) {
      if (index === activeIndex) return;
      activeIndex = index;

      var items = bodyEl.children;
      for (var i = 0; i < items.length; i++) {
        if (i === index) {
          items[i].classList.add('is-active');
        } else {
          items[i].classList.remove('is-active');
        }
      }
      revealActive();
    }

    // 大纲比面板高时，跟随正文把当前项滚到可视范围内
    function revealActive() {
      var current = bodyEl.children[activeIndex];
      if (!current || bodyEl.scrollHeight <= bodyEl.clientHeight) return;

      var top = current.offsetTop - bodyEl.offsetTop;
      var bottom = top + current.offsetHeight;
      if (top < bodyEl.scrollTop) {
        bodyEl.scrollTop = top;
      } else if (bottom > bodyEl.scrollTop + bodyEl.clientHeight) {
        bodyEl.scrollTop = bottom - bodyEl.clientHeight;
      }
    }

    function updateActive() {
      if (!headings.length) return;

      // 判定线放在粘性栏下方一点：正文里刚滚过判定线的那个标题就是当前节
      var line = stickyHeight() + 32;
      var index = 0;
      for (var i = 0; i < headings.length; i++) {
        if (headings[i].el.getBoundingClientRect().top <= line) {
          index = i;
        } else {
          break;
        }
      }

      // 最后一节往往滚不到判定线，直接按滚动到底处理
      if (atBottom()) index = headings.length - 1;

      setActive(index);
    }

    function goTo(index) {
      var target = headings[index];
      if (!target) return;

      var offset = stickyHeight() + 24;
      var top = window.scrollY + target.el.getBoundingClientRect().top - offset;
      window.scrollTo({ top: top < 0 ? 0 : top, behavior: 'smooth' });
      setActive(index);
    }

    // ===== 重建 / 交互 / 事件 =====

    function refresh() {
      var host = document.querySelector(rootSelector);
      var next = host ? collect(host) : [];
      var nextSignature = next.map(function (item) {
        return item.level + ':' + item.text;
      }).join('\u0000');

      headings = next;

      if (nextSignature !== signature) {
        signature = nextSignature;
        renderItems();
      }

      layout();
      activeIndex = -1; // 文章可能已换过，强制重算一次高亮
      updateActive();
    }

    function setCollapsed(value) {
      collapsed = value;
      writeCollapsed(value);
      layout();

      toggleEl.setAttribute('aria-expanded', value ? 'false' : 'true');
      var focusTarget = value ? tab : toggleEl;
      if (!focusTarget.hidden && focusTarget.focus) {
        try {
          focusTarget.focus({ preventScroll: true });
        } catch (err) {
          focusTarget.focus();
        }
      }
    }

    toggleEl.addEventListener('click', function () {
      setCollapsed(true);
    });

    tab.addEventListener('click', function () {
      setCollapsed(false);
    });

    bodyEl.addEventListener('click', function (event) {
      var target = event.target && event.target.closest ? event.target.closest('.wxmd-outline-item') : null;
      if (!target) return;
      goTo(Number(target.getAttribute('data-index')));
    });

    window.addEventListener('scroll', function () {
      if (scrollFrame) return;
      scrollFrame = requestAnimationFrame(function () {
        scrollFrame = null;
        updateActive();
      });
    }, { passive: true });

    window.addEventListener('resize', function () {
      if (resizeTimer) clearTimeout(resizeTimer);
      resizeTimer = setTimeout(function () {
        resizeTimer = null;
        layout();
        updateActive();
      }, RESIZE_DELAY);
    });

    // 字体/图片加载完可能改变正文宽度，重新量一次
    window.addEventListener('load', function () {
      layout();
      updateActive();
    });

    var observer = new MutationObserver(function () {
      if (rebuildTimer) clearTimeout(rebuildTimer);
      rebuildTimer = setTimeout(function () {
        rebuildTimer = null;
        refresh();
      }, REBUILD_DELAY);
    });
    observer.observe(document.querySelector(rootSelector) || document.body, {
      childList: true,
      subtree: true
    });

    setCollapsed(collapsed);
    refresh();

    instance = { refresh: refresh, collapse: setCollapsed };
    return instance;
  }

  window.WXMDOutline = { mount: mount };
})();
