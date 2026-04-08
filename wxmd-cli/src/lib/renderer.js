/**
 * Markdown 渲染引擎
 * 统一复用 frontend/render-core.js
 */

const renderCore = require('../../../frontend/render-core.js');
const MarkdownIt = require('markdown-it');
const hljs = require('highlight.js');
const { JSDOM } = require('jsdom');
const { STYLES, resolveStyleKey } = require('./styles');

function parseHtmlInNode(html) {
  return new JSDOM(html).window.document;
}

/**
 * 渲染 Markdown 为 HTML
 * @param {string} markdown - Markdown 内容
 * @param {string} styleKey - 样式键名
 * @returns {string} HTML 内容
 */
function renderMarkdown(markdown, styleKey = 'wechat-default') {
  const resolvedStyleKey = resolveStyleKey(styleKey);
  return renderCore.renderMarkdown(markdown, {
    styleKey: resolvedStyleKey,
    styles: STYLES,
    markdownit: MarkdownIt,
    hljs,
    parseHtml: parseHtmlInNode,
  });
}

/**
 * 预处理 Markdown 内容
 * @param {string} content
 * @returns {string}
 */
function preprocessMarkdown(content) {
  return renderCore.preprocessMarkdown(content);
}

/**
 * 过滤引用角标
 * @param {string} content
 * @returns {string}
 */
function stripCitationMarkers(content) {
  return renderCore.stripCitationMarkers(content);
}

/**
 * 应用内联样式到 HTML
 * @param {string} html
 * @param {string} styleKey
 * @returns {string}
 */
function applyInlineStyles(html, styleKey) {
  const resolvedStyleKey = resolveStyleKey(styleKey);
  return renderCore.applyInlineStyles(html, {
    styleKey: resolvedStyleKey,
    styles: STYLES,
    parseHtml: parseHtmlInNode,
  });
}

module.exports = {
  renderMarkdown,
  preprocessMarkdown,
  stripCitationMarkers,
  applyInlineStyles,
};
