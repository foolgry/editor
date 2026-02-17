package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Share 分享数据结构
type Share struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Style     string    `json:"style"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateShareRequest 创建分享请求
type CreateShareRequest struct {
	Content string `json:"content"`
	Style   string `json:"style"`
}

var db *sql.DB

func main() {
	// 初始化数据库
	if err := initDB(); err != nil {
		log.Fatal("数据库初始化失败:", err)
	}
	defer db.Close()

	// 设置路由
	mux := http.NewServeMux()
	
	// API 路由
	mux.HandleFunc("/api/share", handleCreateShare)
	mux.HandleFunc("/api/share/", handleGetShare)
	
	// 分享页面路由
	mux.HandleFunc("/s/", handleSharePage)
	
	// 静态文件服务
	staticDir := filepath.Join("..", ".")
	fileServer := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", fileServer)

	// 配置 CORS
	handler := enableCORS(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("服务器启动在 http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// initDB 初始化 SQLite 数据库
func initDB() error {
	var err error
	
	// 确保数据库目录存在
	dbDir := "./data"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("创建数据库目录失败: %w", err)
	}
	
	dbPath := filepath.Join(dbDir, "shares.db")
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	// 创建分享表
	createTableSQL := `CREATE TABLE IF NOT EXISTS shares (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		style TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	// 创建索引
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_shares_created_at ON shares(created_at);")
	if err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	return nil
}

// enableCORS 启用跨域支持
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleCreateShare 创建分享
func handleCreateShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "请求格式错误",
		})
		return
	}

	// 验证内容
	if strings.TrimSpace(req.Content) == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "内容不能为空",
		})
		return
	}

	// 生成唯一 ID
	shareID := generateShareID()
	now := time.Now()

	// 插入数据库
	_, err := db.Exec(
		"INSERT INTO shares (id, content, style, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		shareID, req.Content, req.Style, now, now,
	)
	if err != nil {
		log.Printf("插入分享失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "保存分享失败",
		})
		return
	}

	// 返回分享信息
	share := Share{
		ID:        shareID,
		Content:   req.Content,
		Style:     req.Style,
		CreatedAt: now,
		UpdatedAt: now,
	}

	respondJSON(w, http.StatusCreated, share)
}

// handleGetShare 获取分享内容
func handleGetShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 提取分享 ID
	path := strings.TrimPrefix(r.URL.Path, "/api/share/")
	shareID := strings.Split(path, "/")[0]

	if shareID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "分享 ID 不能为空",
		})
		return
	}

	// 查询数据库
	var share Share
	err := db.QueryRow(
		"SELECT id, content, style, created_at, updated_at FROM shares WHERE id = ?",
		shareID,
	).Scan(&share.ID, &share.Content, &share.Style, &share.CreatedAt, &share.UpdatedAt)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "分享不存在",
		})
		return
	}
	if err != nil {
		log.Printf("查询分享失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "获取分享失败",
		})
		return
	}

	respondJSON(w, http.StatusOK, share)
}

// handleSharePage 分享页面
func handleSharePage(w http.ResponseWriter, r *http.Request) {
	// 提取分享 ID
	shareID := strings.TrimPrefix(r.URL.Path, "/s/")
	shareID = strings.Split(shareID, "/")[0]

	if shareID == "" {
		http.Error(w, "分享 ID 不能为空", http.StatusBadRequest)
		return
	}

	// 查询数据库
	var share Share
	err := db.QueryRow(
		"SELECT id, content, style, created_at, updated_at FROM shares WHERE id = ?",
		shareID,
	).Scan(&share.ID, &share.Content, &share.Style, &share.CreatedAt, &share.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "分享不存在或已过期", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("查询分享失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
		return
	}

	// 返回分享页面 HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(generateSharePageHTML(share)))
}

// generateShareID 生成分享 ID
func generateShareID() string {
	return uuid.New().String()[:8]
}

// respondJSON 返回 JSON 响应
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// generateSharePageHTML 生成分享页面 HTML
func generateSharePageHTML(share Share) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>分享的文章 - 公众号排版器</title>
  <meta name="description" content="通过公众号排版器分享的文章">
  
  <!-- Markdown 渲染库 -->
  <script src="https://cdn.jsdelivr.net/npm/markdown-it@14.0.0/dist/markdown-it.min.js"></script>
  
  <!-- 代码高亮库 -->
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/styles/atom-one-dark.min.css">
  <script src="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/es/highlight.min.js"></script>
  
  <!-- Vue.js -->
  <script src="https://cdn.jsdelivr.net/npm/vue@3.4.15/dist/vue.global.prod.js"></script>
  
  <style>
    :root {
      --color-primary: #000;
      --color-secondary: #666;
      --color-tertiary: #999;
      --color-accent: #0066FF;
      --color-bg: #FAFAFA;
      --color-surface: #FFF;
      --color-border: #E0E0E0;
      --font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, sans-serif;
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
    
    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 16px 24px;
      border-bottom: 1px solid var(--color-border);
      background: var(--color-surface);
      position: sticky;
      top: 0;
      z-index: 100;
    }
    
    .logo {
      font-size: 16px;
      font-weight: 600;
      color: var(--color-primary);
      letter-spacing: -0.02em;
      display: flex;
      align-items: center;
      gap: 8px;
      text-decoration: none;
    }
    
    .logo:hover {
      opacity: 0.8;
    }
    
    .header-actions {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    
    .style-badge {
      padding: 6px 12px;
      background: var(--color-bg);
      border: 1px solid var(--color-border);
      border-radius: 6px;
      font-size: 13px;
      color: var(--color-secondary);
    }
    
    .btn {
      padding: 8px 16px;
      background: var(--color-accent);
      color: white;
      border: none;
      border-radius: 6px;
      font-size: 13px;
      font-weight: 500;
      cursor: pointer;
      text-decoration: none;
      display: inline-flex;
      align-items: center;
      gap: 6px;
      transition: opacity 0.2s;
    }
    
    .btn:hover {
      opacity: 0.9;
    }
    
    .btn-secondary {
      background: var(--color-surface);
      color: var(--color-primary);
      border: 1px solid var(--color-border);
    }
    
    .btn-secondary:hover {
      background: var(--color-bg);
    }
    
    .content {
      max-width: 800px;
      margin: 0 auto;
      padding: 40px 24px;
    }
    
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
      border-radius: 50%%;
      animation: spin 1s linear infinite;
      margin-bottom: 16px;
    }
    
    @keyframes spin {
      to { transform: rotate(360deg); }
    }
    
    .footer {
      text-align: center;
      padding: 40px 24px;
      border-top: 1px solid var(--color-border);
      margin-top: 60px;
    }
    
    .footer-text {
      font-size: 13px;
      color: var(--color-tertiary);
    }
    
    .footer a {
      color: var(--color-accent);
      text-decoration: none;
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
    
    @media (max-width: 768px) {
      .header {
        padding: 12px 16px;
        flex-wrap: wrap;
        gap: 8px;
      }
      
      .content {
        padding: 24px 16px;
      }
      
      .btn-text {
        display: none;
      }
    }
  </style>
</head>
<body>
  <div id="app">
    <header class="header">
      <a href="/" class="logo">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="20" height="20">
          <rect width="32" height="32" rx="6" fill="#0066FF"/>
          <path d="M6 10 L6 22 L9 22 L9 15 L13 20 L17 15 L17 22 L20 22 L20 10 L16 10 L13 14 L10 10 Z" fill="white"/>
          <rect x="6" y="24" width="20" height="2" rx="1" fill="white" opacity="0.7"/>
        </svg>
        <span>公众号排版器</span>
      </a>
      <div class="header-actions">
        <span class="style-badge">%%s</span>
        <button class="btn" @click="copyContent">
          <span v-if="copySuccess">✓</span>
          <span class="btn-text">{{ copySuccess ? '已复制' : '复制内容' }}</span>
        </button>
        <a href="/" class="btn btn-secondary">
          <span>创作</span>
        </a>
      </div>
    </header>
    
    <main class="content">
      <div v-if="loading" class="loading">
        <div class="spinner"></div>
        <p>正在加载内容...</p>
      </div>
      
      <div v-else-if="error" class="error">
        <div class="error-icon">😕</div>
        <h3>{{ error }}</h3>
        <p style="margin-top: 8px; font-size: 14px;">
          <a href="/" style="color: var(--color-accent);">返回首页</a>
        </p>
      </div>
      
      <div v-else v-html="renderedContent"></div>
    </main>
    
    <footer class="footer">
      <p class="footer-text">
        由 <a href="/" target="_blank">公众号排版器</a> 生成 · 
        <a href="/" target="_blank">去创作</a>
      </p>
    </footer>
  </div>

  <script src="/styles.js"></script>
  <script>
    const { createApp } = Vue;
    
    createApp({
      data() {
        return {
          loading: true,
          error: null,
          renderedContent: '',
          markdownContent: %q,
          style: %q,
          copySuccess: false,
          md: null
        };
      },
      
      mounted() {
        this.initMarkdown();
        this.renderContent();
      },
      
      methods: {
        initMarkdown() {
          this.md = window.markdownit({
            html: true,
            linkify: true,
            typographer: false,
            highlight: function (str, lang) {
              const dots = '<div style="display: flex; align-items: center; gap: 6px; padding: 10px 12px; background: #2a2c33; border-bottom: 1px solid #1e1f24;"><span style="width: 12px; height: 12px; border-radius: 50%%; background: #ff5f56;"></span><span style="width: 12px; height: 12px; border-radius: 50%%; background: #ffbd2e;"></span><span style="width: 12px; height: 12px; border-radius: 50%%; background: #27c93f;"></span></div>';
              
              let codeContent = '';
              if (lang && typeof hljs !== 'undefined' && hljs.getLanguage(lang)) {
                try {
                  codeContent = hljs.highlight(str, { language: lang }).value;
                } catch (__) {
                  codeContent = str;
                }
              } else {
                codeContent = str;
              }
              
              return '<div style="margin: 20px 0; border-radius: 8px; overflow: hidden; background: #383a42; box-shadow: 0 2px 8px rgba(0,0,0,0.15);">' + dots + '<div style="padding: 16px; overflow-x: auto; background: #383a42;"><code style="display: block; color: #abb2bf; font-family: \\'SF Mono\\', Monaco, \\'Cascadia Code\\', Consolas, monospace; font-size: 14px; line-height: 1.6; white-space: pre;">' + codeContent + '</code></div></div>';
            }
          });
        },
        
        renderContent() {
          try {
            let html = this.md.render(this.markdownContent);
            html = this.applyInlineStyles(html);
            this.renderedContent = html;
            this.loading = false;
          } catch (err) {
            console.error('渲染失败:', err);
            this.error = '内容渲染失败';
            this.loading = false;
          }
        },
        
        applyInlineStyles(html) {
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
        
        async copyContent() {
          try {
            // 获取渲染后的内容
            const content = this.renderedContent;
            
            // 创建 Blob 用于复制
            const blob = new Blob([content], { type: 'text/html' });
            const clipboardItem = new ClipboardItem({ 'text/html': blob });
            
            await navigator.clipboard.write([clipboardItem]);
            
            this.copySuccess = true;
            setTimeout(() => {
              this.copySuccess = false;
            }, 2000);
          } catch (err) {
            console.error('复制失败:', err);
            // 降级方案
            const textarea = document.createElement('textarea');
            textarea.value = this.markdownContent;
            document.body.appendChild(textarea);
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);
            
            this.copySuccess = true;
            setTimeout(() => {
              this.copySuccess = false;
            }, 2000);
          }
        }
      }
    }).mount('#app');
  </script>
</body>
</html>`, share.Style, share.Style, share.Content, share.Style)
}
