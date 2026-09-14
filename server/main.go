package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	Content   string `json:"content"`
	Style     string `json:"style"`
	Project   string `json:"project"`
	ProjectID string `json:"projectId"`
}

// ShareDetail 分享详情（含归属项目信息），用于创建/挂载/移除等接口响应；
// 嵌入 Share 后 JSON 平铺，向后兼容原有字段
type ShareDetail struct {
	Share
	ProjectID   *string `json:"projectId"`
	ProjectName *string `json:"projectName"`
	ProjectURL  *string `json:"projectUrl"`
	DeepURL     *string `json:"deepUrl"`
}

// Project 项目数据结构（聚合多篇分享的可选容器）
type Project struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	CreatorTokenID sql.NullString `json:"creatorTokenId"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// ProjectListItem 项目列表项（含文章数聚合与创建者信息）
type ProjectListItem struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	ShareCount       int        `json:"shareCount"`
	LastShareAt      *time.Time `json:"lastShareAt"`
	CreatorTokenID   *string    `json:"creatorTokenId"`
	CreatorTokenName *string    `json:"creatorTokenName"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// AuthContext 认证上下文：主密码（站长）或令牌身份
type AuthContext struct {
	IsOwner   bool   // 主密码（站长）
	TokenID   string // 令牌身份时有效
	TokenName string
}

var db *sql.DB

var citeMarkerPattern = regexp.MustCompile(`\x{E200}cite\x{E202}[^\x{E201}]*\x{E201}`)

var listPagePassword string

const listPasswordHeader = "X-List-Password"

type ShareListItem struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Style            string    `json:"style"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	ProjectID        *string   `json:"projectId"`
	ProjectName      *string   `json:"projectName"`
	CreatorTokenID   *string   `json:"creatorTokenId"`
	CreatorTokenName *string   `json:"creatorTokenName"`
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
	mux.HandleFunc("/api/upload", handleUploadImage)

	// 项目与令牌 API
	mux.HandleFunc("/api/projects", handleProjects)
	mux.HandleFunc("/api/projects/", handleProjectByID)
	mux.HandleFunc("/api/tokens", handleTokens)
	mux.HandleFunc("/api/tokens/", handleTokenByID)
	mux.HandleFunc("/api/auth/me", handleAuthMe)

	// 上传的图片
	uploadsDir := resolveUploadsDir()
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))))

	// 分享页面路由
	mux.HandleFunc("/s/", handleSharePage)
	mux.HandleFunc("/list", handleListPage)
	mux.HandleFunc("/p/", handleProjectPage)

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

	// 创建令牌表（多令牌认证，只存哈希，明文仅在签发时返回一次）
	tokensTableSQL := `CREATE TABLE IF NOT EXISTS tokens (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		revoked INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err = db.Exec(tokensTableSQL); err != nil {
		return fmt.Errorf("创建令牌表失败: %w", err)
	}

	// 创建项目表（项目名在同一创建者内部唯一；站长创建时 creator_token_id 为 NULL）
	projectsTableSQL := `CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		creator_token_id TEXT REFERENCES tokens(id),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(creator_token_id, name)
	);`
	if _, err = db.Exec(projectsTableSQL); err != nil {
		return fmt.Errorf("创建项目表失败: %w", err)
	}

	// shares 表增量迁移：project_id / creator_token_id / sort_order（SQLite 不支持
	// ADD COLUMN IF NOT EXISTS，需先用 PRAGMA table_info 检查已有列，保证幂等）
	existingCols, err := getTableColumns("shares")
	if err != nil {
		return fmt.Errorf("读取 shares 表结构失败: %w", err)
	}
	newColumns := map[string]string{
		"project_id":       "TEXT",
		"creator_token_id": "TEXT",
		"sort_order":       "INTEGER",
	}
	for col, colType := range newColumns {
		if existingCols[col] {
			continue
		}
		if _, err = db.Exec(fmt.Sprintf("ALTER TABLE shares ADD COLUMN %s %s;", col, colType)); err != nil {
			return fmt.Errorf("为 shares 表添加列 %s 失败: %w", col, err)
		}
	}

	// 创建索引
	indexSQLs := []string{
		"CREATE INDEX IF NOT EXISTS idx_shares_created_at ON shares(created_at);",
		"CREATE INDEX IF NOT EXISTS idx_shares_project_id ON shares(project_id);",
		"CREATE INDEX IF NOT EXISTS idx_shares_creator_token_id ON shares(creator_token_id);",
	}
	for _, s := range indexSQLs {
		if _, err = db.Exec(s); err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}

	return nil
}

// getTableColumns 读取表已有的列名集合（用于增量迁移的幂等判断）
func getTableColumns(table string) (map[string]bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
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

// handleCreateShare 创建分享（不带 project 保持匿名公开；带 project 需要令牌或主密码）
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

	// 归入项目：请求一旦涉及项目就必须携带凭证（令牌或主密码）。
	// projectId 精确匹配已有项目（编辑器/管理页下拉场景）；project 按名字解析、
	// 不存在自动创建（ADR-0003 的 Agent 语义）。两者都给时 projectId 优先。
	projectName := strings.TrimSpace(req.Project)
	projectID := strings.TrimSpace(req.ProjectID)
	var project *Project
	var creatorTokenID interface{} // 令牌身份时记录创建者，匿名/主密码时为 NULL

	// 可选认证：带有效令牌发布的独立单篇也记录归属（CONTEXT.md：每篇分享归属于
	// 创建它的那个令牌），无凭证保持真匿名；涉及项目时凭证为必需
	var auth AuthContext
	var authed bool
	if projectName != "" || projectID != "" || r.Header.Get("Authorization") != "" {
		var a AuthContext
		var ok bool
		if a, ok = authenticate(r); ok {
			auth, authed = a, true
		}
	}
	if (projectName != "" || projectID != "") && !authed {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "归入项目需要令牌",
		})
		return
	}

	if projectName != "" || projectID != "" {
		if projectID != "" {
			p, err := getProjectByID(projectID)
			if err == sql.ErrNoRows {
				respondJSON(w, http.StatusNotFound, map[string]string{
					"error": "项目不存在",
				})
				return
			}
			if err != nil {
				log.Printf("查询项目失败: %v", err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "保存分享失败",
				})
				return
			}
			// 项目归创建者所有：挂进他人项目会破坏归属边界，令牌仅能挂本人名下
			if !auth.IsOwner && p.CreatorTokenID.String != auth.TokenID {
				respondJSON(w, http.StatusForbidden, map[string]string{
					"error": "无权归入该项目",
				})
				return
			}
			project = p
		} else {
			p, _, err := resolveOrCreateProject(auth, projectName)
			if err != nil {
				log.Printf("解析项目失败: %v", err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "保存分享失败",
				})
				return
			}
			project = p
		}
	}
	// 令牌身份（无论是否归入项目）都记录创建者；主密码发布的单篇归属站长（NULL）
	if authed && !auth.IsOwner {
		creatorTokenID = auth.TokenID
	}

	// 生成唯一 ID
	shareID := generateShareID()
	now := time.Now()

	var projectIDArg interface{}
	if project != nil {
		projectIDArg = project.ID
	}

	// 插入数据库
	_, err := db.Exec(
		"INSERT INTO shares (id, content, style, created_at, updated_at, project_id, creator_token_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		shareID, req.Content, req.Style, now, now, projectIDArg, creatorTokenID,
	)
	if err != nil {
		log.Printf("插入分享失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "保存分享失败",
		})
		return
	}

	// 返回分享信息（未挂项目时项目相关字段为 null，字段始终存在以保持向后兼容）
	detail := ShareDetail{
		Share: Share{
			ID:        shareID,
			Content:   req.Content,
			Style:     req.Style,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	if project != nil {
		detail.ProjectID = &project.ID
		detail.ProjectName = &project.Name
		projectURL := "/p/" + project.ID
		deepURL := "/p/" + project.ID + "/" + shareID
		detail.ProjectURL = &projectURL
		detail.DeepURL = &deepURL
	}

	respondJSON(w, http.StatusCreated, detail)
}

// resolveOrCreateProject 按认证身份范围解析项目名，不存在则自动创建（幂等语义，
// Agent 无人值守场景核心）。令牌 → creator_token_id=该令牌；主密码 → creator_token_id
// 为 NULL（SQLite 的 UNIQUE 中 NULL 不参与唯一约束，站长项目需在应用层查重防同名）。
// 返回值 created 表示是否为本次新建。
func resolveOrCreateProject(auth AuthContext, name string) (*Project, bool, error) {
	name = strings.TrimSpace(name)

	var p Project
	if auth.IsOwner {
		err := db.QueryRow(
			"SELECT id, name, creator_token_id, created_at, updated_at FROM projects WHERE creator_token_id IS NULL AND name = ?",
			name,
		).Scan(&p.ID, &p.Name, &p.CreatorTokenID, &p.CreatedAt, &p.UpdatedAt)
		if err == nil {
			return &p, false, nil
		}
		if err != sql.ErrNoRows {
			return nil, false, err
		}

		id := generateShareID()
		now := time.Now()
		if _, err := db.Exec(
			"INSERT INTO projects (id, name, creator_token_id, created_at, updated_at) VALUES (?, ?, NULL, ?, ?)",
			id, name, now, now,
		); err != nil {
			return nil, false, err
		}
		return &Project{ID: id, Name: name, CreatedAt: now, UpdatedAt: now}, true, nil
	}

	err := db.QueryRow(
		"SELECT id, name, creator_token_id, created_at, updated_at FROM projects WHERE creator_token_id = ? AND name = ?",
		auth.TokenID, name,
	).Scan(&p.ID, &p.Name, &p.CreatorTokenID, &p.CreatedAt, &p.UpdatedAt)
	if err == nil {
		return &p, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}

	id := generateShareID()
	now := time.Now()
	if _, err := db.Exec(
		"INSERT INTO projects (id, name, creator_token_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, name, auth.TokenID, now, now,
	); err != nil {
		return nil, false, err
	}
	return &Project{
		ID:             id,
		Name:           name,
		CreatorTokenID: sql.NullString{String: auth.TokenID, Valid: true},
		CreatedAt:      now,
		UpdatedAt:      now,
	}, true, nil
}

// getShareDetailByID 查询分享详情（含归属项目信息），找不到返回 sql.ErrNoRows
func getShareDetailByID(shareID string) (*ShareDetail, error) {
	var detail ShareDetail
	var projectID, projectName sql.NullString
	err := db.QueryRow(`
		SELECT s.id, s.content, s.style, s.created_at, s.updated_at, p.id, p.name
		FROM shares s
		LEFT JOIN projects p ON s.project_id = p.id
		WHERE s.id = ?`,
		shareID,
	).Scan(&detail.ID, &detail.Content, &detail.Style, &detail.CreatedAt, &detail.UpdatedAt, &projectID, &projectName)
	if err != nil {
		return nil, err
	}

	if projectID.Valid && projectName.Valid {
		projectURL := "/p/" + projectID.String
		deepURL := "/p/" + projectID.String + "/" + detail.ID
		detail.ProjectID = &projectID.String
		detail.ProjectName = &projectName.String
		detail.ProjectURL = &projectURL
		detail.DeepURL = &deepURL
	}
	return &detail, nil
}

// handleShareByID 按 ID 获取或删除分享，以及挂载（attach）/移出（detach）项目
func handleShareByID(w http.ResponseWriter, r *http.Request) {
	shareID := extractShareID(r.URL.Path, "/api/share/")
	if shareID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "分享 ID 不能为空",
		})
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/api/share/"+shareID)
	switch rest {
	case "":
		switch r.Method {
		case http.MethodGet:
			handleGetShareByID(w, shareID)
		case http.MethodDelete:
			handleDeleteShareByID(w, r, shareID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/attach":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleAttachShare(w, r, shareID)
	case "/detach":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleDetachShare(w, r, shareID)
	default:
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "接口不存在",
		})
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

// handleDeleteShareByID 删除分享：主密码可删任意；令牌仅能删除自己创建的
func handleDeleteShareByID(w http.ResponseWriter, r *http.Request, shareID string) {
	auth, ok := authenticate(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "密码错误或缺失",
		})
		return
	}

	// 先查分享是否存在并做权限校验（令牌只能删自己创建的）
	var creatorTokenID sql.NullString
	err := db.QueryRow("SELECT creator_token_id FROM shares WHERE id = ?", shareID).Scan(&creatorTokenID)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "分享不存在",
		})
		return
	}
	if err != nil {
		log.Printf("查询分享失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "删除分享失败",
		})
		return
	}
	if !auth.IsOwner && creatorTokenID.String != auth.TokenID {
		respondJSON(w, http.StatusForbidden, map[string]string{
			"error": "无权删除该分享",
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

// canOperateShare 分享操作权限：主密码任意；无创建者（匿名/主密码发布）的分享仅主密码
// 可操作；令牌仅可操作本人创建的分享（ADR-0002：令牌管自己名下的，主密码管一切）
func canOperateShare(auth AuthContext, creatorTokenID sql.NullString) bool {
	if auth.IsOwner {
		return true
	}
	if !creatorTokenID.Valid {
		return false
	}
	return creatorTokenID.String == auth.TokenID
}

// getShareCreatorTokenID 查询分享的创建者令牌，找不到返回 sql.ErrNoRows
func getShareCreatorTokenID(shareID string) (sql.NullString, error) {
	var creatorTokenID sql.NullString
	err := db.QueryRow("SELECT creator_token_id FROM shares WHERE id = ?", shareID).Scan(&creatorTokenID)
	return creatorTokenID, err
}

// resolveAttachProject 解析挂载目标：优先按 projectId 精确匹配（管理页下拉场景，
// 避免按名解析在跨创建者同名时挂错项目），并校验归属（主密码任意；令牌仅本人名下）；
// 否则按名字走 resolveOrCreateProject（ADR-0003 的 Agent 自动创建语义）。
// 返回 false 表示已写响应。
func resolveAttachProject(w http.ResponseWriter, auth AuthContext, projectID, projectName string) (*Project, bool) {
	if projectID != "" {
		project, err := getProjectByID(projectID)
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]string{
				"error": "项目不存在",
			})
			return nil, false
		}
		if err != nil {
			log.Printf("查询项目失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "挂载项目失败",
			})
			return nil, false
		}
		if !auth.IsOwner && project.CreatorTokenID.String != auth.TokenID {
			respondJSON(w, http.StatusForbidden, map[string]string{
				"error": "无权挂载到该项目",
			})
			return nil, false
		}
		return project, true
	}

	project, _, err := resolveOrCreateProject(auth, projectName)
	if err != nil {
		log.Printf("解析项目失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "挂载项目失败",
		})
		return nil, false
	}
	return project, true
}

// handleAttachShare 把分享挂载/移动到项目：body 可带 projectId（精确挂载，不自动创建）
// 或 project（名字，不存在自动创建，按认证身份范围解析）
func handleAttachShare(w http.ResponseWriter, r *http.Request, shareID string) {
	auth, ok := authenticate(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "凭证无效或缺失",
		})
		return
	}

	var req struct {
		Project   string `json:"project"`
		ProjectID string `json:"projectId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "请求格式错误",
		})
		return
	}
	projectName := strings.TrimSpace(req.Project)
	projectID := strings.TrimSpace(req.ProjectID)
	if projectName == "" && projectID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "项目名不能为空",
		})
		return
	}

	creatorTokenID, err := getShareCreatorTokenID(shareID)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "分享不存在",
		})
		return
	}
	if err != nil {
		log.Printf("查询分享失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "挂载项目失败",
		})
		return
	}
	if !canOperateShare(auth, creatorTokenID) {
		respondJSON(w, http.StatusForbidden, map[string]string{
			"error": "无权操作该分享",
		})
		return
	}

	project, ok := resolveAttachProject(w, auth, projectID, projectName)
	if !ok {
		return
	}

	if _, err := db.Exec(
		"UPDATE shares SET project_id = ?, updated_at = ? WHERE id = ?",
		project.ID, time.Now(), shareID,
	); err != nil {
		log.Printf("挂载项目失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "挂载项目失败",
		})
		return
	}

	detail, err := getShareDetailByID(shareID)
	if err != nil {
		log.Printf("查询分享详情失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "挂载项目失败",
		})
		return
	}
	respondJSON(w, http.StatusOK, detail)
}

// handleDetachShare 把分享移出项目（变回独立单篇，/s/<id> 链接保持有效）
func handleDetachShare(w http.ResponseWriter, r *http.Request, shareID string) {
	auth, ok := authenticate(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "凭证无效或缺失",
		})
		return
	}

	creatorTokenID, err := getShareCreatorTokenID(shareID)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "分享不存在",
		})
		return
	}
	if err != nil {
		log.Printf("查询分享失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "移出项目失败",
		})
		return
	}
	if !canOperateShare(auth, creatorTokenID) {
		respondJSON(w, http.StatusForbidden, map[string]string{
			"error": "无权操作该分享",
		})
		return
	}

	if _, err := db.Exec(
		"UPDATE shares SET project_id = NULL, updated_at = ? WHERE id = ?",
		time.Now(), shareID,
	); err != nil {
		log.Printf("移出项目失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "移出项目失败",
		})
		return
	}

	detail, err := getShareDetailByID(shareID)
	if err != nil {
		log.Printf("查询分享详情失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "移出项目失败",
		})
		return
	}
	respondJSON(w, http.StatusOK, detail)
}

// handleListShares 获取分享列表：主密码可见全部；令牌仅可见自己创建的
func handleListShares(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	auth, ok := authenticate(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "密码错误或缺失",
		})
		return
	}

	query := `
		SELECT s.id, s.content, s.style, s.created_at, s.updated_at,
			p.id, p.name, t.id, t.name
		FROM shares s
		LEFT JOIN projects p ON s.project_id = p.id
		LEFT JOIN tokens t ON s.creator_token_id = t.id`
	args := make([]interface{}, 0)
	if !auth.IsOwner {
		query += " WHERE s.creator_token_id = ?"
		args = append(args, auth.TokenID)
	}
	query += " ORDER BY datetime(s.created_at) DESC, s.rowid DESC"

	rows, err := db.Query(query, args...)
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
		var projectID, projectName, creatorTokenID, creatorTokenName sql.NullString

		if err := rows.Scan(&id, &content, &style, &createdAt, &updatedAt, &projectID, &projectName, &creatorTokenID, &creatorTokenName); err != nil {
			log.Printf("扫描分享列表失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "获取分享列表失败",
			})
			return
		}

		items = append(items, ShareListItem{
			ID:               id,
			Title:            extractShareTitle(content),
			Style:            style,
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
			ProjectID:        nullStringPtr(projectID),
			ProjectName:      nullStringPtr(projectName),
			CreatorTokenID:   nullStringPtr(creatorTokenID),
			CreatorTokenName: nullStringPtr(creatorTokenName),
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

// extractShareTitle 从分享内容中提取展示标题（复用列表页的提取规则，空则"无标题"）
func extractShareTitle(content string) string {
	cleanContent := stripCitationMarkers(content)
	title := extractTitleFromMarkdown(cleanContent)
	if title == "" {
		title = extractDescriptionFromMarkdown(cleanContent)
	}
	if title == "" {
		title = "无标题"
	}
	return title
}

// handleProjects 项目集合入口：GET 列表 / POST 创建
func handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListProjects(w, r)
	case http.MethodPost:
		handleCreateProject(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListProjects 列出项目：令牌 → 本人名下；主密码 → 全部
func handleListProjects(w http.ResponseWriter, r *http.Request) {
	auth, ok := authenticate(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "凭证无效或缺失",
		})
		return
	}

	query := `
		SELECT p.id, p.name, p.created_at, p.updated_at, p.creator_token_id, t.name,
			COUNT(s.id) AS share_count, MAX(s.created_at) AS last_share_at
		FROM projects p
		LEFT JOIN tokens t ON p.creator_token_id = t.id
		LEFT JOIN shares s ON s.project_id = p.id`
	args := make([]interface{}, 0)
	if !auth.IsOwner {
		query += " WHERE p.creator_token_id = ?"
		args = append(args, auth.TokenID)
	}
	query += " GROUP BY p.id ORDER BY datetime(p.updated_at) DESC, MAX(s.rowid) DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("查询项目列表失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "获取项目列表失败",
		})
		return
	}
	defer rows.Close()

	items := make([]ProjectListItem, 0)
	for rows.Next() {
		var item ProjectListItem
		var creatorTokenID, creatorTokenName sql.NullString
		// 聚合函数结果丢失列类型声明（driver 返回字符串），需扫描后手工解析时间
		var lastShareAt sql.NullString
		if err := rows.Scan(
			&item.ID, &item.Name, &item.CreatedAt, &item.UpdatedAt, &creatorTokenID, &creatorTokenName,
			&item.ShareCount, &lastShareAt,
		); err != nil {
			log.Printf("扫描项目列表失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "获取项目列表失败",
			})
			return
		}
		item.CreatorTokenID = nullStringPtr(creatorTokenID)
		item.CreatorTokenName = nullStringPtr(creatorTokenName)
		if t, ok := parseSQLiteTime(lastShareAt.String); ok {
			item.LastShareAt = &t
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("遍历项目列表失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "获取项目列表失败",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

// handleCreateProject 显式创建项目（幂等：范围内同名已存在则直接返回既有项目）
func handleCreateProject(w http.ResponseWriter, r *http.Request) {
	auth, ok := authenticate(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "凭证无效或缺失",
		})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "请求格式错误",
		})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "项目名不能为空",
		})
		return
	}

	project, created, err := resolveOrCreateProject(auth, name)
	if err != nil {
		log.Printf("创建项目失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "创建项目失败",
		})
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, projectToResponse(auth, *project))
}

// projectToResponse 构造项目响应（creatorTokenName 需查令牌名，站长创建时为 null）
func projectToResponse(auth AuthContext, project Project) map[string]interface{} {
	creatorTokenID := nullStringPtr(project.CreatorTokenID)
	var creatorTokenName *string
	if creatorTokenID != nil {
		var name string
		if err := db.QueryRow("SELECT name FROM tokens WHERE id = ?", *creatorTokenID).Scan(&name); err == nil {
			creatorTokenName = &name
		} else if err != sql.ErrNoRows {
			log.Printf("查询令牌名失败: %v", err)
		}
	}
	return map[string]interface{}{
		"id":               project.ID,
		"name":             project.Name,
		"creatorTokenId":   creatorTokenID,
		"creatorTokenName": creatorTokenName,
		"createdAt":        project.CreatedAt,
		"updatedAt":        project.UpdatedAt,
	}
}

// getProjectByID 按 ID 查询项目，找不到返回 sql.ErrNoRows
func getProjectByID(projectID string) (*Project, error) {
	var p Project
	err := db.QueryRow(
		"SELECT id, name, creator_token_id, created_at, updated_at FROM projects WHERE id = ?",
		projectID,
	).Scan(&p.ID, &p.Name, &p.CreatorTokenID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// listProjectShares 查询项目内分享目录（按创建时间倒序，含标题提取，不含 content 全文）
func listProjectShares(projectID string) ([]ShareListItem, error) {
	rows, err := db.Query(`
		SELECT id, content, style, created_at, updated_at
		FROM shares
		WHERE project_id = ?
		ORDER BY datetime(created_at) DESC, rowid DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ShareListItem, 0)
	for rows.Next() {
		var item ShareListItem
		var content, style string
		if err := rows.Scan(&item.ID, &content, &style, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Title = extractShareTitle(content)
		item.Style = style
		items = append(items, item)
	}
	return items, rows.Err()
}

// handleProjectByID 单个项目入口：GET 公开详情 / PATCH 重命名 / DELETE 删除
func handleProjectByID(w http.ResponseWriter, r *http.Request) {
	projectID := extractShareID(r.URL.Path, "/api/projects/")
	if projectID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "项目 ID 不能为空",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetProjectByID(w, projectID)
	case http.MethodPatch:
		handleRenameProject(w, r, projectID)
	case http.MethodDelete:
		handleDeleteProject(w, r, projectID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetProjectByID 项目详情（公开，供项目页前端渲染）
func handleGetProjectByID(w http.ResponseWriter, projectID string) {
	project, err := getProjectByID(projectID)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "项目不存在",
		})
		return
	}
	if err != nil {
		log.Printf("查询项目失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "获取项目失败",
		})
		return
	}

	shares, err := listProjectShares(projectID)
	if err != nil {
		log.Printf("查询项目文章失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "获取项目失败",
		})
		return
	}

	shareItems := make([]map[string]interface{}, 0, len(shares))
	for _, s := range shares {
		shareItems = append(shareItems, map[string]interface{}{
			"id":        s.ID,
			"title":     s.Title,
			"createdAt": s.CreatedAt,
			"updatedAt": s.UpdatedAt,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":        project.ID,
		"name":      project.Name,
		"createdAt": project.CreatedAt,
		"updatedAt": project.UpdatedAt,
		"shares":    shareItems,
	})
}

// requireProjectAccess 校验认证与项目操作权限（主密码任意；令牌仅本人名下），
// 返回项目与认证上下文；已写响应时返回 false
func requireProjectAccess(w http.ResponseWriter, r *http.Request, projectID string) (AuthContext, *Project, bool) {
	auth, ok := authenticate(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "凭证无效或缺失",
		})
		return auth, nil, false
	}

	project, err := getProjectByID(projectID)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "项目不存在",
		})
		return auth, nil, false
	}
	if err != nil {
		log.Printf("查询项目失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "操作项目失败",
		})
		return auth, nil, false
	}

	if !auth.IsOwner && project.CreatorTokenID.String != auth.TokenID {
		respondJSON(w, http.StatusForbidden, map[string]string{
			"error": "无权操作该项目",
		})
		return auth, nil, false
	}
	return auth, project, true
}

// handleRenameProject 重命名项目（刷新 updated_at）
func handleRenameProject(w http.ResponseWriter, r *http.Request, projectID string) {
	auth, project, ok := requireProjectAccess(w, r, projectID)
	if !ok {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "请求格式错误",
		})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "项目名不能为空",
		})
		return
	}

	// 范围内同名查重（不同项目 ID）：查重范围按**目标项目的创建者**圈定（而非当前
	// 认证身份——站长重命名令牌的项目时也要在该令牌范围内查重）。令牌场景由 UNIQUE
	// 约束兜底，站长场景（creator_token_id 为 NULL）SQLite 的 UNIQUE 对 NULL 不生效，
	// 必须应用层防重
	dupCheck := "SELECT id FROM projects WHERE name = ? AND id != ? AND creator_token_id = ?"
	dupArgs := []interface{}{name, projectID, project.CreatorTokenID.String}
	if !project.CreatorTokenID.Valid {
		dupCheck = "SELECT id FROM projects WHERE name = ? AND id != ? AND creator_token_id IS NULL"
		dupArgs = dupArgs[:2]
	}
	var dupID string
	err := db.QueryRow(dupCheck, dupArgs...).Scan(&dupID)
	if err == nil {
		respondJSON(w, http.StatusConflict, map[string]string{
			"error": "同名项目已存在",
		})
		return
	}
	if err != sql.ErrNoRows {
		log.Printf("查询同名项目失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "重命名项目失败",
		})
		return
	}

	now := time.Now()
	if _, err := db.Exec(
		"UPDATE projects SET name = ?, updated_at = ? WHERE id = ?",
		name, now, projectID,
	); err != nil {
		log.Printf("重命名项目失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "重命名项目失败",
		})
		return
	}

	project.Name = name
	project.UpdatedAt = now
	respondJSON(w, http.StatusOK, projectToResponse(auth, *project))
}

// handleDeleteProject 删除项目：先解除文章归属（变回独立单篇，链接保持有效），再删容器
func handleDeleteProject(w http.ResponseWriter, r *http.Request, projectID string) {
	_, project, ok := requireProjectAccess(w, r, projectID)
	if !ok {
		return
	}

	result, err := db.Exec("UPDATE shares SET project_id = NULL WHERE project_id = ?", projectID)
	if err != nil {
		log.Printf("解除文章归属失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "删除项目失败",
		})
		return
	}
	releasedCount, err := result.RowsAffected()
	if err != nil {
		log.Printf("获取解除归属行数失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "删除项目失败",
		})
		return
	}

	if _, err := db.Exec("DELETE FROM projects WHERE id = ?", projectID); err != nil {
		log.Printf("删除项目失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "删除项目失败",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":            project.ID,
		"releasedCount": releasedCount,
		"message":       fmt.Sprintf("项目已删除，%d 篇文章变为独立分享", releasedCount),
	})
}

// requireOwner 令牌管理仅限站长主密码；返回 false 时已写响应
func requireOwner(w http.ResponseWriter, r *http.Request) (AuthContext, bool) {
	auth, ok := authenticate(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "凭证无效或缺失",
		})
		return auth, false
	}
	if !auth.IsOwner {
		respondJSON(w, http.StatusForbidden, map[string]string{
			"error": "仅站长主密码可管理令牌",
		})
		return auth, false
	}
	return auth, true
}

// handleTokens 令牌集合入口：GET 列表 / POST 签发（均仅主密码）
func handleTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListTokens(w, r)
	case http.MethodPost:
		handleCreateToken(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListTokens 令牌列表（不返回哈希）
func handleListTokens(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireOwner(w, r); !ok {
		return
	}

	rows, err := db.Query("SELECT id, name, revoked, created_at FROM tokens ORDER BY datetime(created_at) ASC")
	if err != nil {
		log.Printf("查询令牌列表失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "获取令牌列表失败",
		})
		return
	}
	defer rows.Close()

	type tokenItem struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Revoked   bool      `json:"revoked"`
		CreatedAt time.Time `json:"createdAt"`
	}
	items := make([]tokenItem, 0)
	for rows.Next() {
		var item tokenItem
		var revoked int
		if err := rows.Scan(&item.ID, &item.Name, &revoked, &item.CreatedAt); err != nil {
			log.Printf("扫描令牌列表失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "获取令牌列表失败",
			})
			return
		}
		item.Revoked = revoked != 0
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("遍历令牌列表失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "获取令牌列表失败",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

// handleCreateToken 签发令牌（明文仅在本次响应返回一次，库里只存 sha256 哈希）
func handleCreateToken(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireOwner(w, r); !ok {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "请求格式错误",
		})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "令牌名不能为空",
		})
		return
	}

	tokenPlain, err := generateTokenString()
	if err != nil {
		log.Printf("生成令牌失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "生成令牌失败",
		})
		return
	}

	tokenID := generateShareID()
	now := time.Now()
	if _, err := db.Exec(
		"INSERT INTO tokens (id, name, token_hash, revoked, created_at) VALUES (?, ?, ?, 0, ?)",
		tokenID, name, sha256Hex(tokenPlain), now,
	); err != nil {
		log.Printf("写入令牌失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "生成令牌失败",
		})
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":        tokenID,
		"name":      name,
		"token":     tokenPlain,
		"createdAt": now,
	})
}

// handleTokenByID 单个令牌入口：DELETE 吊销（软删除，仅主密码）
func handleTokenByID(w http.ResponseWriter, r *http.Request) {
	tokenID := extractShareID(r.URL.Path, "/api/tokens/")
	if tokenID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "令牌 ID 不能为空",
		})
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, ok := requireOwner(w, r); !ok {
		return
	}

	result, err := db.Exec("UPDATE tokens SET revoked = 1 WHERE id = ?", tokenID)
	if err != nil {
		log.Printf("吊销令牌失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "吊销令牌失败",
		})
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		log.Printf("获取吊销影响行数失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "吊销令牌失败",
		})
		return
	}
	if affected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "令牌不存在",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":      tokenID,
		"revoked": true,
	})
}

// handleAuthMe 探测当前凭证身份（令牌或站长主密码）
func handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	auth, ok := authenticate(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "凭证无效或缺失",
		})
		return
	}

	if auth.IsOwner {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"type":    "owner",
			"name":    "站长",
			"isOwner": true,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"type":    "token",
		"name":    auth.TokenName,
		"isOwner": false,
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

// getShareByID 查询分享全量数据（含 content），找不到返回 sql.ErrNoRows
func getShareByID(shareID string) (*Share, error) {
	var share Share
	err := db.QueryRow(
		"SELECT id, content, style, created_at, updated_at FROM shares WHERE id = ?",
		shareID,
	).Scan(&share.ID, &share.Content, &share.Style, &share.CreatedAt, &share.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &share, nil
}

// handleProjectPage 项目聚合页面路由：/p/<pid> 与 /p/<pid>/<sid>
func handleProjectPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/p/")
	parts := strings.Split(rest, "/")
	pid := parts[0]
	sid := ""
	if len(parts) > 1 {
		sid = parts[1]
	}

	if pid == "" {
		http.Error(w, "项目不存在", http.StatusNotFound)
		return
	}

	project, err := getProjectByID(pid)
	if err == sql.ErrNoRows {
		http.Error(w, "项目不存在", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("查询项目失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
		return
	}

	shares, err := listProjectShares(pid)
	if err != nil {
		log.Printf("查询项目文章失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
		return
	}

	// 深链兜底：sid 不存在或已不属于该项目时，302 跳转 /s/<sid>（链接存活原则）
	var currentShare *Share
	if sid != "" {
		belongs := false
		for _, s := range shares {
			if s.ID == sid {
				belongs = true
				break
			}
		}
		if !belongs {
			http.Redirect(w, r, "/s/"+sid, http.StatusFound)
			return
		}
		share, err := getShareByID(sid)
		if err != nil {
			log.Printf("查询分享失败: %v", err)
			http.Error(w, "服务器错误", http.StatusInternalServerError)
			return
		}
		currentShare = share
	} else if len(shares) > 0 {
		// 不带 sid 默认展示最新一篇
		share, err := getShareByID(shares[0].ID)
		if err != nil {
			log.Printf("查询分享失败: %v", err)
			http.Error(w, "服务器错误", http.StatusInternalServerError)
			return
		}
		currentShare = share
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(generateProjectPageHTML(*project, shares, currentShare)))
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

// authenticate 统一认证入口：
//  1. Authorization: Bearer <credential>：先查 tokens 表（哈希匹配且未吊销）→ 令牌身份；
//     查不到再与主密码等值比对 → 站长身份；
//  2. X-List-Password: <password>：仅主密码 → 站长身份（兼容现有客户端）。
func authenticate(r *http.Request) (AuthContext, bool) {
	authHeader := r.Header.Get("Authorization")
	if credential := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")); authHeader != "" && credential != "" {
		tokenHash := sha256Hex(credential)
		var id, name string
		err := db.QueryRow(
			"SELECT id, name FROM tokens WHERE token_hash = ? AND revoked = 0",
			tokenHash,
		).Scan(&id, &name)
		if err == nil {
			return AuthContext{IsOwner: false, TokenID: id, TokenName: name}, true
		}
		if err != sql.ErrNoRows {
			log.Printf("查询令牌失败: %v", err)
			return AuthContext{}, false
		}
		// 非令牌，尝试主密码（站长）
		if credential == listPagePassword {
			return AuthContext{IsOwner: true}, true
		}
		return AuthContext{}, false
	}

	// 兼容现有 X-List-Password 头：仅主密码 → 站长
	if password := strings.TrimSpace(r.Header.Get(listPasswordHeader)); password != "" && password == listPagePassword {
		return AuthContext{IsOwner: true}, true
	}

	return AuthContext{}, false
}

// generateTokenString 生成令牌明文：wmt_ + 32 字节随机数的 base64url 编码
func generateTokenString() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "wmt_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

// sha256Hex 计算字符串的 SHA-256 十六进制摘要（令牌只存哈希）
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// nullStringPtr 将 sql.NullString 转为可空 JSON 指针
func nullStringPtr(ns sql.NullString) *string {
	if ns.Valid {
		value := ns.String
		return &value
	}
	return nil
}

// parseSQLiteTime 解析 SQLite 返回的时间字符串（聚合函数结果丢失列类型，
// driver 以字符串返回，需手工解析；覆盖 go-sqlite3 写入与 CURRENT_TIMESTAMP 两种格式）
func parseSQLiteTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	formats := []string{
		"2006-01-02 15:04:05.999999999-07:00", // go-sqlite3 写入 time.Time 的格式
		"2006-01-02 15:04:05.999999999",
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05", // CURRENT_TIMESTAMP 默认格式（UTC）
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// 允许上传的图片类型
var allowedImageExts = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
}

const maxUploadSize = 10 << 20 // 10MB

// resolveUploadsDir 上传图片存储目录，随 SQLite 数据目录一起持久化
func resolveUploadsDir() string {
	return filepath.Join(".", "data", "uploads")
}

// handleUploadImage 上传图片，返回可访问的 URL
func handleUploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "请求解析失败或文件超过 10MB 限制",
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "缺少 file 字段",
		})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if _, ok := allowedImageExts[ext]; !ok {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "不支持的图片格式，仅支持 png/jpg/jpeg/gif/webp/svg",
		})
		return
	}

	uploadsDir := resolveUploadsDir()
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Printf("创建上传目录失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "保存图片失败",
		})
		return
	}

	filename := uuid.New().String() + ext
	dst, err := os.Create(filepath.Join(uploadsDir, filename))
	if err != nil {
		log.Printf("创建图片文件失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "保存图片失败",
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		log.Printf("写入图片失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "保存图片失败",
		})
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{
		"url": "/uploads/" + filename,
	})
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
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  <link rel="alternate icon" href="/favicon.svg">
  <link rel="mask-icon" href="/favicon.svg" color="#0066FF">
  
  <!-- 代码高亮样式（自托管） -->
  <link rel="stylesheet" href="/lib/vendor/atom-one-dark.min.css">

  <!-- 核心库（自托管 + defer 按序执行，不阻塞首屏渲染） -->
  <script src="/lib/vendor/markdown-it.min.js" defer></script>
  <script src="/lib/vendor/highlight.min.js" defer></script>
  <script src="/lib/vendor/vue.global.prod.js" defer></script>

  <!-- mermaid（3MB+）改为按需加载，仅当文章含图表时才引入（见下方 renderMermaid） -->
  <script src="/lib/lazy-loader.js" defer></script>
  
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

  <script src="/lib/lazy-loader.js" defer></script>
  <script src="/render-core.js" defer></script>
  <script src="/styles.js" defer></script>
  <script>
    // 库脚本均带 defer，会在 DOMContentLoaded 之前按文档顺序执行完毕，
    // 因此在 DOMContentLoaded 回调里启动应用时 Vue / markdown-it / hljs 已就绪
    document.addEventListener('DOMContentLoaded', function() {
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
    });
  </script>
</body>
</html>`

	replacer := strings.NewReplacer(
		"__WX_EDITOR_TITLE__", html.EscapeString(title),
		"__WX_EDITOR_DESCRIPTION__", html.EscapeString(description),
		"__WX_EDITOR_MARKDOWN_CONTENT__", inlineJSONString(cleanContent),
		"__WX_EDITOR_STYLE__", inlineJSONString(share.Style),
	)

	return replacer.Replace(pageTemplate)
}

func inlineJSONString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
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
