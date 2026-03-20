/**
 * Markdown 文本工具
 */
(() => {
const EMPHASIS_MARKERS = new Set([
  0x2A, // *
  0x5F, // _
  0x7E  // ~
]);

function isCjkLetter(charCode) {
  if (!charCode || charCode < 0) {
    return false;
  }

  return (
    (charCode >= 0x3400 && charCode <= 0x4DBF) ||  // CJK Unified Ideographs Extension A
    (charCode >= 0x4E00 && charCode <= 0x9FFF) ||  // CJK Unified Ideographs
    (charCode >= 0xF900 && charCode <= 0xFAFF) ||  // CJK Compatibility Ideographs
    (charCode >= 0xFF01 && charCode <= 0xFF60) ||  // Full-width ASCII variants
    (charCode >= 0xFF61 && charCode <= 0xFF9F) ||  // Half-width Katakana
    (charCode >= 0xFFA0 && charCode <= 0xFFDC)     // Full-width Latin letters
  );
}

  window.WechatEditorModules = window.WechatEditorModules || {};
  window.WechatEditorModules.EMPHASIS_MARKERS = EMPHASIS_MARKERS;
  window.WechatEditorModules.isCjkLetter = isCjkLetter;
})();
