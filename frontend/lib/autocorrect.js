/**
 * AutoCorrect - JavaScript Implementation
 * 自动修复 CJK（中日韩）与英文混排时的空格问题
 * 
 * 基于 huacnlee/autocorrect 的核心规则简化实现
 */

(function(global) {
  'use strict';

  // CJK Unicode 范围
  const CJK_REGEX = /[\u4e00-\u9fa5\u3040-\u309f\u30a0-\u30ff\u3100-\u312f\uac00-\ud7af]/;
  
  // 英文字母和数字
  const ALPHANUM_REGEX = /[a-zA-Z0-9]/;
  
  // 中文标点
  const CJK_PUNCTUATION = /[\u3000-\u303f\uff00-\uffef]/;
  
  // 英文标点（在中文语境下应该转为中文标点）
  const EN_PUNCT_TO_CJK = {
    ',': '，',
    '.': '。',
    '!': '！',
    '?': '？',
    ':': '：',
    ';': '；',
    '(': '（',
    ')': '）',
    '[': '【',
    ']': '】',
    '<': '《',
    '>': '》',
    '"': '"',
    "'": ''
  };

  /**
   * 判断字符是否是 CJK
   */
  function isCJK(char) {
    if (!char) return false;
    return CJK_REGEX.test(char);
  }

  /**
   * 判断字符是否是英文或数字
   */
  function isAlphaNum(char) {
    if (!char) return false;
    return ALPHANUM_REGEX.test(char);
  }

  /**
   * 判断字符是否是空格
   */
  function isSpace(char) {
    return char === ' ' || char === '\t';
  }

  /**
   * 保护 Markdown 语法片段，避免格式化时破坏标记本身
   */
  function protectMarkdownSyntax(text) {
    const protectedSegments = [];
    const syntaxRegex = /!\[[^\]\n]*\](?:\([^\)\n]*\)|\[[^\]\n]*\])?|\[[^\]\n]*\](?:\([^\)\n]*\)|\[[^\]\n]*\])/g;
    const protectedText = text.replace(syntaxRegex, (match) => {
      const token = `\uE000MD${protectedSegments.length}\uE001`;
      protectedSegments.push(match);
      return token;
    });

    return { protectedText, protectedSegments };
  }

  function restoreMarkdownSyntax(text, protectedSegments) {
    return text.replace(/\uE000MD(\d+)\uE001/g, (_, index) => {
      return protectedSegments[Number(index)] || _;
    });
  }

  /**
   * 格式化文本 - 核心函数
   */
  function format(text) {
    if (!text || typeof text !== 'string') {
      return text;
    }

    const { protectedText, protectedSegments } = protectMarkdownSyntax(text);

    let result = '';
    let prevChar = '';

    for (let i = 0; i < protectedText.length; i++) {
      let char = protectedText[i];
      let nextChar = protectedText[i + 1] || '';

      // 在 CJK 和英文/数字之间添加空格
      if (isCJK(char) && isAlphaNum(nextChar)) {
        result += char + ' ';
      } else if (isAlphaNum(char) && isCJK(nextChar)) {
        result += char + ' ';
      } else {
        // 检查是否需要转换标点符号（CJK 后面的英文标点转为中文标点）
        if (isCJK(prevChar) && EN_PUNCT_TO_CJK[char] && !isCJK(nextChar)) {
          result += EN_PUNCT_TO_CJK[char];
        } else {
          result += char;
        }
      }

      prevChar = char;
    }

    // 清理连续空格
    result = result.replace(/  +/g, ' ');
    
    // 清理行首行尾空格
    result = result.replace(/\n /g, '\n').replace(/ \n/g, '\n');

    return restoreMarkdownSyntax(result, protectedSegments);
  }

  /**
   * 根据文件类型格式化
   */
  function formatFor(text, filetype) {
    // 简化实现，直接调用 format
    // 实际可以根据 filetype 调整行为
    return format(text);
  }

  // 导出 API
  const AutoCorrect = {
    format: format,
    formatFor: formatFor,
    isLoaded: true
  };

  // 兼容 ES Module 和全局变量
  if (typeof module !== 'undefined' && module.exports) {
    module.exports = AutoCorrect;
  } else {
    global.autocorrect = AutoCorrect;
    global.AutoCorrect = AutoCorrect;
  }

})(typeof window !== 'undefined' ? window : global);
