package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
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

var citeMarkerPattern = regexp.MustCompile(`\x{E200}cite\x{E202}[^\x{E201}]*\x{E201}`)

var listPagePassword string

const listPasswordHeader = "X-List-Password"

type ShareListItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Style     string    `json:"style"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func main() {
	// 加载 .env 文件（尝试多个路径：项目根目录、当前目录）
	for _, p := range []string{"../.env", ".env"} {
		godotenv.Load(p)
	}

	listPagePassword = os.Getenv("LIST_PAGE_PASSWORD")
	if listPagePassword == "" {
		log.Fatal("环境变量 LIST_PAGE_PASSWORD 未设置，请检查 .env 文件")
	}

	// 初始化数据库
	if err := initDB(); err != nil {
		log.Fatal("数据库初始化失败:", err)
	}
	defer db.Close()

	// 设置路由
	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/api/share", handleCreateShare)
	mux.HandleFunc("/api/share/", handleShareByID)
	mux.HandleFunc("/api/shares", handleListShares)

	// 分享页面路由
	mux.HandleFunc("/s/", handleSharePage)
	mux.HandleFunc("/list", handleListPage)

	// 静态文件服务
	staticDir := resolveStaticDir()
	log.Printf("静态文件目录: %s", staticDir)
	fileServer := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", fileServer)

	// CORS 中间件
	handler := corsMiddleware(mux)

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

// CORS 中间件 - 处理跨域请求
func corsMiddleware(next http.Handler) http.Handler {
	// 从环境变量读取允许的来源
	allowedOrigins := parseCORSOrigins()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// 检查是否允许的域名
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		// 允许的头部和方法
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-List-Password")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24小时缓存预检结果

		// 处理预检请求
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

// handleShareByID 按 ID 获取或删除分享
func handleShareByID(w http.ResponseWriter, r *http.Request) {
	shareID := extractShareID(r.URL.Path, "/api/share/")
	if shareID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "分享 ID 不能为空",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetShareByID(w, shareID)
	case http.MethodDelete:
		handleDeleteShareByID(w, r, shareID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetShareByID(w http.ResponseWriter, shareID string) {
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

func handleDeleteShareByID(w http.ResponseWriter, r *http.Request, shareID string) {
	if !isListAuthorized(r) {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "密码错误或缺失",
		})
		return
	}

	result, err := db.Exec("DELETE FROM shares WHERE id = ?", shareID)
	if err != nil {
		log.Printf("删除分享失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "删除分享失败",
		})
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Printf("获取删除影响行数失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "删除分享失败",
		})
		return
	}
	if affected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "分享不存在",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"id":      shareID,
		"message": "删除成功",
	})
}

// handleListShares 获取全部分享列表（需要密码）
func handleListShares(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isListAuthorized(r) {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "密码错误或缺失",
		})
		return
	}

	rows, err := db.Query(`
		SELECT id, content, style, created_at, updated_at
		FROM shares
		ORDER BY datetime(created_at) DESC
	`)
	if err != nil {
		log.Printf("查询分享列表失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "获取分享列表失败",
		})
		return
	}
	defer rows.Close()

	items := make([]ShareListItem, 0)
	for rows.Next() {
		var id string
		var content string
		var style string
		var createdAt time.Time
		var updatedAt time.Time

		if err := rows.Scan(&id, &content, &style, &createdAt, &updatedAt); err != nil {
			log.Printf("扫描分享列表失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "获取分享列表失败",
			})
			return
		}

		cleanContent := stripCitationMarkers(content)
		title := extractTitleFromMarkdown(cleanContent)
		if title == "" {
			title = extractDescriptionFromMarkdown(cleanContent)
		}
		if title == "" {
			title = "无标题"
		}

		items = append(items, ShareListItem{
			ID:        id,
			Title:     title,
			Style:     style,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("遍历分享列表失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "获取分享列表失败",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"count": len(items),
	})
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

func handleListPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(generateListPageHTML()))
}

// generateShareID 生成分享 ID
func generateShareID() string {
	return uuid.New().String()[:8]
}

func extractShareID(path string, prefix string) string {
	trimmed := strings.TrimPrefix(path, prefix)
	return strings.Split(trimmed, "/")[0]
}

func isListAuthorized(r *http.Request) bool {
	password := strings.TrimSpace(r.Header.Get(listPasswordHeader))
	return password == listPagePassword
}

// respondJSON 返回 JSON 响应
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// generateSharePageHTML 生成分享页面 HTML
func generateSharePageHTML(share Share) string {
	cleanContent := stripCitationMarkers(share.Content)

	// 提取标题：从 Markdown 内容中找第一个 # 开头的标题
	title := extractTitleFromMarkdown(cleanContent)
	if title == "" {
		title = "分享的文章"
	}

	// 提取描述：从内容中提取前 150 个字符作为描述
	description := extractDescriptionFromMarkdown(cleanContent)
	if description == "" {
		description = "通过公众号排版器分享的文章"
	}

	const pageTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>__WX_EDITOR_TITLE__</title>
  <meta name="description" content="__WX_EDITOR_DESCRIPTION__">
  <link rel="icon" type="image/jpeg" href="/favicon.jpg">
  <link rel="shortcut icon" href="/favicon.jpg">
  <link rel="apple-touch-icon" href="/favicon.jpg">
  
  <!-- Markdown 渲染库 -->
  <script src="https://cdn.jsdelivr.net/npm/markdown-it@14.0.0/dist/markdown-it.min.js"></script>
  
  <!-- 代码高亮库 -->
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/styles/atom-one-dark.min.css">
  <script src="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/es/highlight.min.js"></script>
  
  <!-- Mermaid 图表库 -->
  <script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
  
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
      border-radius: 50%;
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
  </div>

  <script src="/render-core.js"></script>
  <script src="/styles.js"></script>
  <script>
    const { createApp } = Vue;
    
    createApp({
      data() {
        return {
          loading: true,
          error: null,
          renderedContent: '',
          markdownContent: __WX_EDITOR_MARKDOWN_CONTENT__,
          style: __WX_EDITOR_STYLE__,
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
            const processedContent = renderCore && typeof renderCore.preprocessMarkdown === 'function'
              ? renderCore.preprocessMarkdown(this.markdownContent)
              : this.markdownContent;

            let html = this.md.render(processedContent);
            html = this.applyInlineStyles(html);
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
        
        renderMermaid() {
          if (typeof mermaid !== 'undefined') {
            try {
              mermaid.initialize({
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
                mermaid.run({
                  querySelector: '.mermaid'
                });
              }
            } catch (err) {
              console.error('Mermaid 渲染失败:', err);
            }
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
</html>`

	replacer := strings.NewReplacer(
		"__WX_EDITOR_TITLE__", html.EscapeString(title),
		"__WX_EDITOR_DESCRIPTION__", html.EscapeString(description),
		"__WX_EDITOR_MARKDOWN_CONTENT__", strconv.Quote(cleanContent),
		"__WX_EDITOR_STYLE__", strconv.Quote(share.Style),
	)

	return replacer.Replace(pageTemplate)
}

func generateListPageHTML() string {
	const pageTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>分享列表</title>
  <style>
    * { box-sizing: border-box; }
    body {
      margin: 0;
      padding: 24px;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
      background: #f5f7fb;
      color: #111827;
    }
    .container {
      max-width: 960px;
      margin: 0 auto;
      background: #fff;
      border: 1px solid #e5e7eb;
      border-radius: 12px;
      overflow: hidden;
    }
    .header {
      padding: 20px 24px;
      border-bottom: 1px solid #e5e7eb;
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 12px;
      flex-wrap: wrap;
    }
    .title {
      margin: 0;
      font-size: 20px;
      font-weight: 700;
    }
    .muted {
      color: #6b7280;
      font-size: 13px;
    }
    .auth {
      padding: 32px 24px;
      border-bottom: 1px solid #e5e7eb;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .row {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
    }
    input[type="password"] {
      width: min(360px, 100%);
      height: 40px;
      border: 1px solid #d1d5db;
      border-radius: 8px;
      padding: 0 12px;
      font-size: 14px;
    }
    button {
      height: 40px;
      border: none;
      background: #111827;
      color: #fff;
      border-radius: 8px;
      padding: 0 16px;
      font-size: 14px;
      cursor: pointer;
    }
    button:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
    .btn-delete {
      height: 32px;
      padding: 0 10px;
      border-radius: 6px;
      background: #dc2626;
      font-size: 13px;
    }
    .error {
      color: #b91c1c;
      font-size: 13px;
    }
    .list-wrap {
      padding: 0 24px 24px;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      table-layout: fixed;
    }
    th, td {
      border-bottom: 1px solid #e5e7eb;
      text-align: left;
      padding: 12px 8px;
      font-size: 14px;
      vertical-align: top;
      word-break: break-word;
    }
    th {
      color: #374151;
      background: #f9fafb;
      font-weight: 600;
    }
    a {
      color: #2563eb;
      text-decoration: none;
    }
    a:hover {
      text-decoration: underline;
    }
    .status {
      padding: 16px 24px;
      font-size: 14px;
      color: #4b5563;
    }
    @media (max-width: 768px) {
      body { padding: 12px; }
      .header, .auth, .list-wrap, .status { padding-left: 14px; padding-right: 14px; }
      th, td { font-size: 13px; padding: 10px 6px; }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1 class="title">分享列表</h1>
      <div class="muted">需要密码访问</div>
    </div>

    <div id="auth" class="auth">
      <div class="muted">请输入密码后加载全部分享记录（密码会保存在当前浏览器，无过期）。</div>
      <div class="row">
        <input id="pwd" type="password" placeholder="请输入密码" autocomplete="off" />
        <button id="submitBtn">进入</button>
      </div>
      <div id="authError" class="error"></div>
    </div>

    <div id="status" class="status">等待输入密码</div>
    <div id="listWrap" class="list-wrap"></div>
  </div>

  <script>
    const PASSWORD_STORAGE_KEY = "wx-editor-list-password";
    const PASSWORD_HEADER_KEY = "X-List-Password";
    const authEl = document.getElementById("auth");
    const pwdEl = document.getElementById("pwd");
    const submitBtnEl = document.getElementById("submitBtn");
    const authErrorEl = document.getElementById("authError");
    const statusEl = document.getElementById("status");
    const listWrapEl = document.getElementById("listWrap");
    let currentPassword = "";

    function escapeHTML(text) {
      return String(text || "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
    }

    function formatDate(value) {
      if (!value) return "-";
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) return "-";
      return date.toLocaleString("zh-CN", { hour12: false });
    }

    function renderList(items) {
      if (!Array.isArray(items) || items.length === 0) {
        listWrapEl.innerHTML = "<div class='muted' style='padding-top: 12px;'>暂无分享记录</div>";
        return;
      }

      const rows = items.map((item) => {
        const id = escapeHTML(item.id);
        const title = escapeHTML(item.title || "无标题");
        const style = escapeHTML(item.style || "-");
        const createdAt = escapeHTML(formatDate(item.createdAt));
        const updatedAt = escapeHTML(formatDate(item.updatedAt));
        return "<tr>" +
          "<td><a href='/s/" + id + "' target='_blank' rel='noopener noreferrer'>" + id + "</a></td>" +
          "<td>" + title + "</td>" +
          "<td>" + style + "</td>" +
          "<td>" + createdAt + "</td>" +
          "<td>" + updatedAt + "</td>" +
          "<td><button class='btn-delete' data-id='" + id + "'>删除</button></td>" +
          "</tr>";
      }).join("");

      listWrapEl.innerHTML = "<table>" +
        "<thead><tr><th style='width:120px;'>ID</th><th>标题</th><th style='width:140px;'>样式</th><th style='width:170px;'>创建时间</th><th style='width:170px;'>更新时间</th><th style='width:92px;'>操作</th></tr></thead>" +
        "<tbody>" + rows + "</tbody>" +
        "</table>";
    }

    async function loadList(password) {
      statusEl.textContent = "正在加载分享列表...";
      authErrorEl.textContent = "";
      submitBtnEl.disabled = true;

      try {
        const response = await fetch("/api/shares", {
          headers: {
            [PASSWORD_HEADER_KEY]: password
          }
        });

        const data = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(data.error || "加载失败");
        }

        localStorage.setItem(PASSWORD_STORAGE_KEY, password);
        currentPassword = password;
        authEl.style.display = "none";
        const count = Number(data.count || 0);
        statusEl.textContent = "共 " + count + " 条记录";
        renderList(data.items || []);
      } catch (error) {
        statusEl.textContent = "加载失败";
        authErrorEl.textContent = error && error.message ? error.message : "密码错误或网络异常";
      } finally {
        submitBtnEl.disabled = false;
      }
    }

    async function deleteShare(id, buttonEl) {
      if (!currentPassword) {
        authErrorEl.textContent = "密码不存在，请重新输入";
        authEl.style.display = "flex";
        return;
      }
      if (!window.confirm("确认删除该分享记录？删除后不可恢复。")) {
        return;
      }

      const prevText = buttonEl.textContent;
      buttonEl.disabled = true;
      buttonEl.textContent = "删除中...";
      authErrorEl.textContent = "";

      try {
        const response = await fetch("/api/share/" + encodeURIComponent(id), {
          method: "DELETE",
          headers: {
            [PASSWORD_HEADER_KEY]: currentPassword
          }
        });

        const data = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(data.error || "删除失败");
        }

        await loadList(currentPassword);
      } catch (error) {
        authErrorEl.textContent = error && error.message ? error.message : "删除失败";
      } finally {
        buttonEl.disabled = false;
        buttonEl.textContent = prevText;
      }
    }

    submitBtnEl.addEventListener("click", () => {
      const password = (pwdEl.value || "").trim();
      if (!password) {
        authErrorEl.textContent = "请输入密码";
        return;
      }
      loadList(password);
    });

    pwdEl.addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        submitBtnEl.click();
      }
    });

    listWrapEl.addEventListener("click", (event) => {
      const target = event.target;
      if (!(target instanceof HTMLButtonElement)) {
        return;
      }
      const id = target.getAttribute("data-id");
      if (!id) {
        return;
      }
      deleteShare(id, target);
    });

    const cachedPassword = localStorage.getItem(PASSWORD_STORAGE_KEY);
    if (cachedPassword) {
      pwdEl.value = cachedPassword;
      loadList(cachedPassword);
    }
  </script>
</body>
</html>`

	return pageTemplate
}

func stripCitationMarkers(content string) string {
	return citeMarkerPattern.ReplaceAllString(content, "")
}

// extractTitleFromMarkdown 从 Markdown 内容中提取标题
func extractTitleFromMarkdown(content string) string {
	// 查找第一个 # 开头的标题
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 匹配 # 标题、## 标题等
		if strings.HasPrefix(line, "#") {
			// 去掉 # 和空格
			title := strings.TrimLeft(line, "#")
			title = strings.TrimSpace(title)
			if title != "" {
				return title
			}
		}
	}
	return ""
}

// extractDescriptionFromMarkdown 从 Markdown 内容中提取描述
func extractDescriptionFromMarkdown(content string) string {
	// 移除代码块
	content = removeCodeBlocks(content)

	// 按行分割，找第一个非空、非标题、非特殊标记的行
	lines := strings.Split(content, "\n")
	var description strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行
		if line == "" {
			continue
		}

		// 跳过标题行
		if strings.HasPrefix(line, "#") {
			continue
		}

		// 跳过特殊 Markdown 标记
		if strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") ||
			strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") ||
			strings.HasPrefix(line, ">") || strings.HasPrefix(line, "|") ||
			strings.HasPrefix(line, "```") {
			continue
		}

		// 移除 Markdown 链接标记 [text](url) -> text
		line = removeMarkdownLinks(line)

		// 移除其他 Markdown 标记
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "*", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.ReplaceAll(line, "_", "")
		line = strings.ReplaceAll(line, "`", "")

		if line != "" {
			description.WriteString(line)
			// 如果描述已经足够长，截断并添加省略号
			if description.Len() >= 150 {
				truncated := description.String()[:150]
				// 确保不在单词中间截断
				lastSpace := strings.LastIndex(truncated, " ")
				if lastSpace > 100 {
					return truncated[:lastSpace] + "..."
				}
				return truncated + "..."
			}
			description.WriteString(" ")
		}
	}

	result := strings.TrimSpace(description.String())
	if len(result) > 150 {
		return result[:150] + "..."
	}
	return result
}

// removeCodeBlocks 移除 Markdown 代码块
func removeCodeBlocks(content string) string {
	var result strings.Builder
	inCodeBlock := false
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if !inCodeBlock {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return result.String()
}

// removeMarkdownLinks 移除 Markdown 链接，保留链接文本
func removeMarkdownLinks(line string) string {
	// 简单的正则替换：将 [text](url) 替换为 text
	for {
		start := strings.Index(line, "[")
		if start == -1 {
			break
		}
		end := strings.Index(line[start:], "]")
		if end == -1 {
			break
		}
		end += start

		// 检查后面是否有 (url)
		if end+1 < len(line) && line[end+1] == '(' {
			urlEnd := strings.Index(line[end+1:], ")")
			if urlEnd != -1 {
				urlEnd += end + 1
				// 提取文本并替换
				text := line[start+1 : end]
				line = line[:start] + text + line[urlEnd+1:]
				continue
			}
		}
		break
	}
	return line
}

// parseCORSOrigins 从环境变量 CORS_ORIGINS 读取允许的来源列表
func parseCORSOrigins() []string {
	env := os.Getenv("CORS_ORIGINS")
	if env == "" {
		return []string{"http://localhost:8080", "http://localhost:3000"}
	}
	origins := strings.Split(env, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}
	return origins
}

// resolveStaticDir 自动探测前端静态目录，兼容本地开发与容器运行
func resolveStaticDir() string {
	candidates := []string{"../frontend", "./frontend"}
	for _, dir := range candidates {
		indexFile := filepath.Join(dir, "index.html")
		if info, err := os.Stat(indexFile); err == nil && !info.IsDir() {
			return dir
		}
	}
	// 回退到默认值，保持行为可预测
	return "../frontend"
}
