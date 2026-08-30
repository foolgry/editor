/**
 * 按需加载器
 * mermaid / turndown / html2canvas 体积大且多数场景用不到，
 * 首屏不再加载，首次调用相关功能时才动态插入 <script>。
 * 所有路径使用绝对路径，编辑器首页与 /s/ 分享页均可复用。
 */
(function(global) {
  'use strict';

  var loadedScripts = {};

  function loadScript(src) {
    if (loadedScripts[src]) {
      return loadedScripts[src];
    }

    loadedScripts[src] = new Promise(function(resolve, reject) {
      var script = document.createElement('script');
      script.src = src;
      script.async = true;
      script.onload = function() {
        resolve();
      };
      script.onerror = function() {
        delete loadedScripts[src];
        reject(new Error('脚本加载失败: ' + src));
      };
      document.head.appendChild(script);
    });

    return loadedScripts[src];
  }

  var WXMDLazy = {
    loadScript: loadScript,

    loadMermaid: function() {
      if (global.mermaid) {
        return Promise.resolve(global.mermaid);
      }
      return loadScript('/lib/vendor/mermaid.min.js').then(function() {
        return global.mermaid;
      });
    },

    loadTurndown: function() {
      if (global.TurndownService) {
        return Promise.resolve(global.TurndownService);
      }
      return loadScript('/lib/vendor/turndown.js').then(function() {
        return global.TurndownService;
      });
    },

    loadHtml2canvas: function() {
      if (global.html2canvas) {
        return Promise.resolve(global.html2canvas);
      }
      return loadScript('/lib/vendor/html2canvas.min.js').then(function() {
        return global.html2canvas;
      });
    }
  };

  global.WXMDLazy = WXMDLazy;
})(typeof window !== 'undefined' ? window : global);
