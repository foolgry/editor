package main

import (
	"net/http"
)

// handleApplyPage 令牌申请页：本站关闭匿名发布后，未持令牌的发布者从这里拿到申请入口。
// 页面公开可访问（申请者此时必然还没有令牌），只放申请途径，不放任何管理能力。
func handleApplyPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(generateApplyPageHTML()))
}

// generateApplyPageHTML 生成申请页 HTML。
// 视觉与 /list 管理页保持一致（同一套 CSS 变量与卡片），静态内容无需转义。
func generateApplyPageHTML() string {
	const pageTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>申请分享令牌</title>
  <style>
    :root {
      --bg: #f5f7fb;
      --card: #ffffff;
      --border: #e5e7eb;
      --text: #111827;
      --muted: #6b7280;
      --link: #2563eb;
    }
    * { box-sizing: border-box; }
    [hidden] { display: none !important; }
    body {
      margin: 0;
      padding: 24px;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
      background: var(--bg);
      color: var(--text);
      line-height: 1.6;
    }
    .container {
      max-width: 640px;
      margin: 0 auto;
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      overflow: hidden;
    }
    .header {
      padding: 20px 24px;
      border-bottom: 1px solid var(--border);
    }
    .title {
      margin: 0;
      font-size: 20px;
      font-weight: 700;
    }
    .body { padding: 24px; }
    .lead {
      margin: 0 0 20px;
      font-size: 14px;
      color: #374151;
    }
    .notice {
      background: #fffbeb;
      border: 1px solid #fde68a;
      color: #92400e;
      border-radius: 8px;
      padding: 10px 12px;
      font-size: 13px;
      margin-bottom: 20px;
    }
    .step {
      display: flex;
      gap: 12px;
      padding: 16px 0;
      border-top: 1px solid var(--border);
    }
    .step-index {
      flex: none;
      width: 24px;
      height: 24px;
      border-radius: 999px;
      background: var(--text);
      color: #fff;
      font-size: 13px;
      font-weight: 600;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .step-main { flex: 1; min-width: 0; }
    .step-title {
      margin: 0 0 6px;
      font-size: 15px;
      font-weight: 600;
    }
    .step-desc {
      margin: 0;
      font-size: 13px;
      color: var(--muted);
    }
    .qr-wrap {
      margin-top: 14px;
      padding: 12px;
      background: #fff;
      border: 1px solid var(--border);
      border-radius: 12px;
      display: inline-block;
    }
    .qr-wrap img {
      display: block;
      width: 220px;
      height: auto;
      border-radius: 6px;
    }
    .qr-caption {
      margin: 8px 0 0;
      font-size: 12px;
      color: var(--muted);
      text-align: center;
    }
    .usage {
      margin: 10px 0 0;
      padding: 0;
      list-style: none;
    }
    .usage li {
      font-size: 13px;
      color: var(--muted);
      padding: 4px 0;
    }
    .usage strong { color: var(--text); font-weight: 600; }
    code {
      background: #f3f4f6;
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 1px 6px;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 12px;
      word-break: break-all;
    }
    pre {
      margin: 10px 0 0;
      background: #f3f4f6;
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px;
      overflow-x: auto;
      font-size: 12px;
    }
    pre code { background: none; border: none; padding: 0; }
    a { color: var(--link); text-decoration: none; }
    a:hover { text-decoration: underline; }
    @media (max-width: 480px) {
      body { padding: 12px; }
      .header, .body { padding-left: 16px; padding-right: 16px; }
      .qr-wrap img { width: 100%; }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1 class="title">申请分享令牌</h1>
    </div>
    <div class="body">
      <p class="lead">
        本站已关闭匿名发布。无论是用网页编辑器，还是用 skill / Agent 脚本发布，
        都需要一枚令牌。令牌用来标识发布者，站长可随时吊销。
      </p>

      <div class="notice">
        令牌只在签发时显示一次，请拿到后立即保存到安全的地方。
      </div>

      <div class="step">
        <div class="step-index">1</div>
        <div class="step-main">
          <h2 class="step-title">添加站长微信</h2>
          <p class="step-desc">扫码添加，备注一下用途（例如「申请发布令牌」），方便快速通过。</p>
          <div class="qr-wrap">
            <img src="/wechat-qr.jpg" alt="站长微信二维码">
            <p class="qr-caption">扫二维码，添加我为好友</p>
          </div>
        </div>
      </div>

      <div class="step">
        <div class="step-index">2</div>
        <div class="step-main">
          <h2 class="step-title">拿到令牌</h2>
          <p class="step-desc">
            站长会为你在管理后台签发一枚令牌（形如 <code>wmt_</code> 开头的一串字符），
            并把令牌发给你。请勿把令牌公开分享给他人——它等同于你的发布凭证。
          </p>
        </div>
      </div>

      <div class="step">
        <div class="step-index">3</div>
        <div class="step-main">
          <h2 class="step-title">用令牌发布</h2>
          <p class="step-desc">两种发布方式都支持同一个令牌：</p>
          <ul class="usage">
            <li>
              <strong>网页编辑器</strong>：点「分享」，在弹层里粘贴令牌并保存，
              浏览器会记住它，之后发布不再需要重复填写。
            </li>
            <li>
              <strong>skill / Agent 脚本</strong>：把令牌放进环境变量
              <code>WXMD_TOKEN</code>，发布和图片上传都会自动带上。
            </li>
          </ul>
          <pre><code>export WXMD_TOKEN=wmt_你拿到的令牌</code></pre>
        </div>
      </div>

      <div class="step">
        <div class="step-index">4</div>
        <div class="step-main">
          <h2 class="step-title">令牌失效了怎么办</h2>
          <p class="step-desc">
            令牌泄露、丢失或需要轮换时，联系站长吊销旧令牌并重新签发一枚。
            已经发布出去的分享链接不受影响，仍然可以正常访问。
          </p>
        </div>
      </div>
    </div>
  </div>
</body>
</html>`

	return pageTemplate
}
