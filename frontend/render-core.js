(function (root, factory) {
  if (typeof module === 'object' && module.exports) {
    module.exports = factory();
    return;
  }
  root.WXMDRenderCore = factory();
})(typeof globalThis !== 'undefined' ? globalThis : this, function () {
  const EMPHASIS_MARKERS = new Set([0x2A, 0x5F, 0x7E]);
  const MERMAID_LANGUAGES = new Set([
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
  ]);

  let scanDelimsPatched = false;

  function isCjkLetter(charCode) {
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
  }

  function escapeHtml(text) {
    return String(text)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function createSafeLeadingPunctuationChecker() {
    const fallbackChars = '「『《〈（【〔〖［｛﹁﹃﹙﹛﹝“‘（';
    const fallbackSet = new Set(
      fallbackChars.split('').map((char) => char.codePointAt(0))
    );

    let unicodeRegex = null;
    try {
      unicodeRegex = new RegExp('[\\p{Ps}\\p{Pi}]', 'u');
    } catch (_error) {
      unicodeRegex = null;
    }

    return function (charCode, marker) {
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
  }

  function patchMarkdownScanner(md) {
    if (!md || !md.inline || !md.inline.State || scanDelimsPatched) {
      return;
    }

    const utils = md.utils;
    const StateInline = md.inline.State;

    if (StateInline.prototype.__wxmdScanDelimsPatched) {
      scanDelimsPatched = true;
      return;
    }

    const allowLeadingPunctuation = createSafeLeadingPunctuationChecker();
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
        pos += 1;
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

      if (marker === 0x5F) {
        if (!isLastWhiteSpace && !isLastPunctChar && isCjkLetter(lastChar)) {
          isLastPunctChar = true;
        }
        if (!isNextWhiteSpace && !isNextPunctChar && isCjkLetter(nextChar)) {
          isNextPunctChar = true;
        }
      }

      const leftFlanking =
        !isNextWhiteSpace && (!isNextPunctChar || isLastWhiteSpace || isLastPunctChar);
      const rightFlanking =
        !isLastWhiteSpace && (!isLastPunctChar || isNextWhiteSpace || isNextPunctChar);

      const canOpen = leftFlanking && (canSplitWord || !rightFlanking || isLastPunctChar);
      const canClose = rightFlanking && (canSplitWord || !leftFlanking || isNextPunctChar);

      return { can_open: canOpen, can_close: canClose, length: count };
    };

    StateInline.prototype.__wxmdScanDelimsPatched = true;
    scanDelimsPatched = true;
  }

  function instantiateMarkdownParser(MarkdownItCtor, config) {
    try {
      return new MarkdownItCtor(config);
    } catch (_error) {
      return MarkdownItCtor(config);
    }
  }

  function loadOptionalNodeDependency(name) {
    if (typeof require !== 'function') {
      return null;
    }

    try {
      return require(name);
    } catch (_error) {
      return null;
    }
  }

  function createMarkdownParser(options) {
    const config = options || {};
    const MarkdownItCtor =
      config.markdownit ||
      (typeof window !== 'undefined' ? window.markdownit : null) ||
      loadOptionalNodeDependency('markdown-it');

    if (!MarkdownItCtor) {
      throw new Error('markdown-it is required to create parser');
    }

    const hljsInstance =
      config.hljs ||
      (typeof window !== 'undefined' ? window.hljs : null) ||
      loadOptionalNodeDependency('highlight.js');

    let md = null;
    md = instantiateMarkdownParser(MarkdownItCtor, {
      html: true,
      linkify: true,
      typographer: false,
      highlight: function (str, lang) {
        if (lang && MERMAID_LANGUAGES.has(lang)) {
          const mermaidSource =
            md && md.utils && typeof md.utils.escapeHtml === 'function'
              ? md.utils.escapeHtml(str)
              : escapeHtml(str);
          return '<div class="mermaid" style="background: #fff; padding: 20px; border-radius: 8px; margin: 20px 0; text-align: center; overflow-x: auto;">' + mermaidSource + '</div>';
        }

        const dots = '<div style="display: flex; align-items: center; gap: 6px; padding: 10px 12px; background: #2a2c33; border-bottom: 1px solid #1e1f24;"><span style="width: 12px; height: 12px; border-radius: 50%; background: #ff5f56;"></span><span style="width: 12px; height: 12px; border-radius: 50%; background: #ffbd2e;"></span><span style="width: 12px; height: 12px; border-radius: 50%; background: #27c93f;"></span></div>';

        let codeContent = '';
        if (
          lang &&
          hljsInstance &&
          typeof hljsInstance.getLanguage === 'function' &&
          typeof hljsInstance.highlight === 'function'
        ) {
          try {
            if (hljsInstance.getLanguage(lang)) {
              codeContent = hljsInstance.highlight(str, { language: lang }).value;
            } else {
              codeContent =
                md && md.utils && typeof md.utils.escapeHtml === 'function'
                  ? md.utils.escapeHtml(str)
                  : escapeHtml(str);
            }
          } catch (_error) {
            codeContent =
              md && md.utils && typeof md.utils.escapeHtml === 'function'
                ? md.utils.escapeHtml(str)
                : escapeHtml(str);
          }
        } else {
          codeContent =
            md && md.utils && typeof md.utils.escapeHtml === 'function'
              ? md.utils.escapeHtml(str)
              : escapeHtml(str);
        }

        return '<div style="margin: 20px 0; border-radius: 8px; overflow: hidden; background: #383a42; box-shadow: 0 2px 8px rgba(0,0,0,0.15);">' + dots + '<div style="padding: 16px; overflow-x: auto; background: #383a42;"><code style="display: block; color: #abb2bf; font-family: \'SF Mono\', Monaco, \'Cascadia Code\', Consolas, monospace; font-size: 14px; line-height: 1.6; white-space: pre;">' + codeContent + '</code></div></div>';
      }
    });

    // 宽表格左右滑动：用 <section class="table-wrapper"> 包裹 <table>
    md.renderer.rules.table_open = function() {
      return '<section class="table-wrapper"><table>';
    };
    md.renderer.rules.table_close = function() {
      return '</table></section>';
    };

    if (config.patchScanner !== false) {
      patchMarkdownScanner(md);
    }

    return md;
  }

  function stripCitationMarkers(content) {
    return String(content || '').replace(/\uE200cite\uE202[^\uE201]*\uE201/g, '');
  }

  function preprocessMarkdown(content) {
    let processed = stripCitationMarkers(content);

    processed = processed.replace(/^[ ]{0,3}(\*[ ]*\*[ ]*\*[\* ]*)[ \t]*$/gm, '***');
    processed = processed.replace(/^[ ]{0,3}(-[ ]*-[ ]*-[- ]*)[ \t]*$/gm, '---');
    processed = processed.replace(/^[ ]{0,3}(_[ ]*_[ ]*_[_ ]*)[ \t]*$/gm, '___');

    processed = processed.replace(/\*\*\s+\*\*/g, ' ');
    processed = processed.replace(/\*{4,}/g, '');
    processed = processed.replace(/__\s+__/g, ' ');
    processed = processed.replace(/_{4,}/g, '');

    processed = processed.replace(/^(\s*(?:\d+\.|-|\*)\s+[^:\n]+)\n\s*:\s*(.+?)$/gm, '$1: $2');
    processed = processed.replace(/^(\s*(?:\d+\.|-|\*)\s+.+?:)\s*\n\s+(.+?)$/gm, '$1 $2');
    processed = processed.replace(/^(\s*(?:\d+\.|-|\*)\s+[^:\n]+)\n:\s*(.+?)$/gm, '$1: $2');
    processed = processed.replace(/^(\s*(?:\d+\.|-|\*)\s+.+?)\n\n(\s+.+?)$/gm, function (match, listItem, continuation) {
      const trimmedContinuation = continuation.trimStart();
      if (/^(>|#{1,6}\s|[-+*]\s|\d+\.\s|```|~~~|\|)/.test(trimmedContinuation)) {
        return match;
      }
      return listItem + ' ' + trimmedContinuation;
    });

    return processed;
  }

  function parseHtml(html, options) {
    if (options && typeof options.parseHtml === 'function') {
      return options.parseHtml(html);
    }

    if (typeof DOMParser !== 'undefined') {
      return new DOMParser().parseFromString(html, 'text/html');
    }

    if (typeof require === 'function') {
      try {
        const jsdom = require('jsdom');
        const dom = new jsdom.JSDOM(html);
        return dom.window.document;
      } catch (_error) {
        // ignore and throw unified error below
      }
    }

    throw new Error('No HTML parser available for render core');
  }

  function resolveStyleConfig(styles, styleKey) {
    if (!styles || typeof styles !== 'object') {
      return null;
    }

    if (styleKey && styles[styleKey] && styles[styleKey].styles) {
      return styles[styleKey].styles;
    }

    if (styles['wechat-default'] && styles['wechat-default'].styles) {
      return styles['wechat-default'].styles;
    }

    const keys = Object.keys(styles);
    if (keys.length > 0 && styles[keys[0]] && styles[keys[0]].styles) {
      return styles[keys[0]].styles;
    }

    return null;
  }

  function groupConsecutiveImages(doc) {
    const body = doc.body;
    const children = Array.from(body.children);
    const imagesToProcess = [];

    children.forEach(function (child, index) {
      if (child.tagName === 'P') {
        const images = child.querySelectorAll('img');
        if (images.length > 0) {
          if (images.length > 1) {
            const group = Array.from(images).map(function (img) {
              return {
                element: child,
                img: img,
                index: index,
                inSameParagraph: true,
                paragraphImageCount: images.length
              };
            });
            imagesToProcess.push.apply(imagesToProcess, group);
          } else {
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
        imagesToProcess.push({
          element: child,
          img: child,
          index: index,
          inSameParagraph: false,
          paragraphImageCount: 1
        });
      }
    });

    const groups = [];
    let currentGroup = [];

    imagesToProcess.forEach(function (item, i) {
      if (i === 0) {
        currentGroup.push(item);
        return;
      }

      const prevItem = imagesToProcess[i - 1];
      let isContinuous = false;

      if (item.index === prevItem.index) {
        isContinuous = true;
      } else if (item.index - prevItem.index === 1) {
        isContinuous = true;
      }

      if (isContinuous) {
        currentGroup.push(item);
      } else {
        if (currentGroup.length > 0) {
          groups.push(currentGroup.slice());
        }
        currentGroup = [item];
      }
    });

    if (currentGroup.length > 0) {
      groups.push(currentGroup);
    }

    groups.forEach(function (group) {
      if (group.length < 2) {
        return;
      }

      const imageCount = group.length;
      const firstElement = group[0].element;

      const gridContainer = doc.createElement('div');
      gridContainer.setAttribute('class', 'image-grid');
      gridContainer.setAttribute('data-image-count', imageCount);

      let gridStyle = '';
      let columns = 2;

      if (imageCount === 2) {
        gridStyle = 'display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin: 20px auto; max-width: 100%; align-items: start;';
        columns = 2;
      } else if (imageCount === 3) {
        gridStyle = 'display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; margin: 20px auto; max-width: 100%; align-items: start;';
        columns = 3;
      } else if (imageCount === 4) {
        gridStyle = 'display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin: 20px auto; max-width: 100%; align-items: start;';
        columns = 2;
      } else {
        gridStyle = 'display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; margin: 20px auto; max-width: 100%; align-items: start;';
        columns = 3;
      }

      gridContainer.setAttribute('style', gridStyle);
      gridContainer.setAttribute('data-columns', columns);

      group.forEach(function (item) {
        const imgWrapper = doc.createElement('div');
        imgWrapper.setAttribute('style', 'width: 100%; height: auto; overflow: hidden;');

        const img = item.img.cloneNode(true);
        img.setAttribute('style', 'width: 100%; height: auto; display: block; border-radius: 8px;');

        imgWrapper.appendChild(img);
        gridContainer.appendChild(imgWrapper);
      });

      firstElement.parentNode.insertBefore(gridContainer, firstElement);

      const elementsToRemove = new Set();
      group.forEach(function (item) {
        elementsToRemove.add(item.element);
      });
      elementsToRemove.forEach(function (element) {
        if (element.parentNode) {
          element.parentNode.removeChild(element);
        }
      });
    });
  }

  function applyInlineStyles(html, options) {
    const config = options || {};
    const style = resolveStyleConfig(config.styles, config.styleKey);

    if (!style) {
      return html;
    }

    const doc = parseHtml(html, config);

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

    groupConsecutiveImages(doc);

    Object.keys(style).forEach(function (selector) {
      if (
        selector === 'pre' ||
        selector === 'code' ||
        selector === 'pre code' ||
        selector === 'container'
      ) {
        return;
      }

      const elements = doc.querySelectorAll(selector);
      elements.forEach(function (el) {
        if (el.tagName === 'IMG' && el.closest('.image-grid')) {
          return;
        }

        const currentStyle = el.getAttribute('style') || '';
        el.setAttribute('style', currentStyle + '; ' + style[selector]);
      });
    });

    // --- 宽表格左右滑动支持（在主题样式注入之后，确保宽度/nowrap 覆盖生效）---
    // 1. 为 .table-wrapper 注入滚动容器样式
    var tableWrappers = doc.querySelectorAll('.table-wrapper');
    tableWrappers.forEach(function(wrapper) {
      var currentStyle = wrapper.getAttribute('style') || '';
      // 基础滚动样式（主题可通过 .table-wrapper 选择器覆盖）
      if (!/overflow-x\s*:/.test(currentStyle)) {
        wrapper.setAttribute('style', currentStyle +
          '; display: block; box-sizing: border-box; width: 100%; max-width: 100%;' +
          ' overflow-x: auto; overflow-y: hidden; -webkit-overflow-scrolling: touch;' +
          ' margin: 20px 0; padding-bottom: 10px;');
      }
    });

    // 2. 修改 table 为固定宽度 + min-width，触发窄屏滚动
    var tables = doc.querySelectorAll('table');
    tables.forEach(function(table) {
      var currentStyle = table.getAttribute('style') || '';
      var cleaned = currentStyle
        .replace(/width\s*:\s*[^;]+;?/gi, '')
        .replace(/min-width\s*:\s*[^;]+;?/gi, '')
        .replace(/max-width\s*:\s*[^;]+;?/gi, '')
        .replace(/table-layout\s*:\s*[^;]+;?/gi, '')
        .replace(/margin\s*:\s*[^;]+;?/gi, '')
        .replace(/;\s*;/g, ';')
        .trim();
      table.setAttribute('style', cleaned +
        '; width: 720px; min-width: 100%; max-width: none; table-layout: auto; margin: 0;');
    });

    // 3. 为 th/td 添加 nowrap 防止内容换行
    var cells = doc.querySelectorAll('th, td');
    cells.forEach(function(cell) {
      var currentStyle = cell.getAttribute('style') || '';
      if (!/white-space\s*:/.test(currentStyle)) {
        cell.setAttribute('style', currentStyle +
          '; white-space: nowrap; word-break: keep-all; overflow-wrap: normal;');
      }
    });
    // --- 宽表格左右滑动支持结束 ---

    const headings = doc.querySelectorAll('h1, h2, h3, h4, h5, h6');
    headings.forEach(function (heading) {
      const inlineNodes = heading.querySelectorAll(headingInlineSelectorList);
      inlineNodes.forEach(function (node) {
        const tag = node.tagName.toLowerCase();
        const override = headingInlineOverrides[tag];
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
    container.setAttribute('style', style.container || '');
    container.innerHTML = doc.body.innerHTML;

    return container.outerHTML;
  }

  function renderMarkdown(markdown, options) {
    const config = options || {};
    const md = config.md || createMarkdownParser(config);
    const processedContent = preprocessMarkdown(markdown);

    let html = md.render(processedContent);
    html = applyInlineStyles(html, config);

    return html;
  }

  return {
    MERMAID_LANGUAGES: Array.from(MERMAID_LANGUAGES),
    createMarkdownParser: createMarkdownParser,
    patchMarkdownScanner: patchMarkdownScanner,
    preprocessMarkdown: preprocessMarkdown,
    stripCitationMarkers: stripCitationMarkers,
    applyInlineStyles: applyInlineStyles,
    renderMarkdown: renderMarkdown
  };
});
