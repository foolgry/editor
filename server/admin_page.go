package main

// generateListPageHTML 生成管理页（/list）HTML。
// 升级要点：
//   - 单输入框双凭证登录：站长主密码或任一有效令牌（Authorization: Bearer），
//     凭证存 localStorage（wx-admin-credential），刷新自动登录；任何请求 401 时
//     清状态回登录页并提示"凭证已失效"；
//   - 三个标签页：分享（挂载/移动项目、移出项目、删除）、项目（新建、查看内含
//     文章、重命名、删除）、令牌（签发/吊销，仅主密码身份可见）；
//   - vanilla JS 无框架，视觉沿用原分享列表页（卡片 + 表格）；所有动态值插入
//     HTML 前一律经过 escapeHTML 转义，不使用模板字符串拼接用户数据。
func generateListPageHTML() string {
	const pageTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>管理后台</title>
  <style>
    :root {
      --bg: #f5f7fb;
      --card: #ffffff;
      --border: #e5e7eb;
      --text: #111827;
      --muted: #6b7280;
      --link: #2563eb;
      --danger: #dc2626;
      --danger-text: #b91c1c;
    }
    * { box-sizing: border-box; }
    /* hidden 属性必须始终生效：.overlay/.identity 等自设 display 会压过 UA 默认规则 */
    [hidden] { display: none !important; }
    body {
      margin: 0;
      padding: 24px;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
      background: var(--bg);
      color: var(--text);
    }
    .container {
      max-width: 1200px;
      margin: 0 auto;
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      overflow: hidden;
    }
    .header {
      padding: 20px 24px;
      border-bottom: 1px solid var(--border);
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
      color: var(--muted);
      font-size: 13px;
    }
    .identity {
      display: flex;
      align-items: center;
      gap: 10px;
      flex-wrap: wrap;
    }
    .badge {
      display: inline-block;
      padding: 2px 10px;
      border-radius: 999px;
      font-size: 12px;
      white-space: nowrap;
    }
    .badge-owner { background: #eef2ff; color: #4338ca; }
    .badge-token { background: #ecfdf5; color: #047857; }
    .badge-active { background: #ecfdf5; color: #047857; }
    .badge-revoked { background: #fef2f2; color: var(--danger-text); }
    .auth {
      padding: 32px 24px;
      border-bottom: 1px solid var(--border);
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .row {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
    }
    input[type="password"], input[type="text"], .input {
      width: min(360px, 100%);
      height: 40px;
      border: 1px solid #d1d5db;
      border-radius: 8px;
      padding: 0 12px;
      font-size: 14px;
      background: #fff;
      color: var(--text);
    }
    .create-row {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin-bottom: 12px;
    }
    .create-row .input { width: min(280px, 100%); }
    button {
      height: 40px;
      border: none;
      background: var(--text);
      color: #fff;
      border-radius: 8px;
      padding: 0 16px;
      font-size: 14px;
      cursor: pointer;
      white-space: nowrap;
    }
    button:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
    .btn-sm {
      height: 30px;
      padding: 0 10px;
      border-radius: 6px;
      font-size: 13px;
    }
    .btn-ghost {
      background: #fff;
      color: var(--text);
      border: 1px solid #d1d5db;
    }
    .btn-danger { background: var(--danger); }
    .error {
      color: var(--danger-text);
      font-size: 13px;
    }
    .feedback {
      padding: 8px 0 12px;
      font-size: 13px;
      color: #4b5563;
      min-height: 20px;
    }
    .feedback.error { color: var(--danger-text); }
    /* 标签页 */
    .tabs {
      display: flex;
      gap: 4px;
      border-bottom: 1px solid var(--border);
      padding: 0 24px;
    }
    .tab-btn {
      background: transparent;
      color: var(--muted);
      height: 44px;
      padding: 0 14px;
      border: none;
      border-bottom: 2px solid transparent;
      border-radius: 0;
      font-size: 14px;
      font-weight: 500;
    }
    .tab-btn.active {
      color: var(--text);
      border-bottom-color: var(--text);
      font-weight: 600;
    }
    .pane { padding: 12px 24px 24px; }
    /* 表格 */
    .table-wrap { overflow-x: auto; }
    table {
      width: 100%;
      border-collapse: collapse;
    }
    th, td {
      border-bottom: 1px solid var(--border);
      text-align: left;
      padding: 12px 8px;
      font-size: 14px;
      vertical-align: top;
      word-break: break-word;
      white-space: normal;
    }
    th {
      color: #374151;
      background: #f9fafb;
      font-weight: 600;
      white-space: nowrap;
    }
    a {
      color: var(--link);
      text-decoration: none;
    }
    a:hover { text-decoration: underline; }
    .ops {
      display: flex;
      gap: 6px;
      flex-wrap: wrap;
    }
    /* 分享列表：列宽由 colgroup 决定，标题列吃掉剩余宽度并保底 200px */
    .shares-table { table-layout: fixed; min-width: 1150px; }
    .shares-table td.nowrap, .shares-table th.nowrap { white-space: nowrap; }
    /* 项目详情展开行 */
    .detail-cell {
      background: #f9fafb;
      padding: 12px 16px;
    }
    .detail-item {
      display: flex;
      align-items: center;
      gap: 12px;
      flex-wrap: wrap;
      padding: 6px 0;
      border-bottom: 1px dashed var(--border);
      font-size: 13px;
    }
    .detail-item:last-child { border-bottom: none; }
    .detail-item .detail-title { flex: 1 1 240px; }
    /* 弹层 */
    .overlay {
      position: fixed;
      inset: 0;
      background: rgba(17, 24, 39, 0.45);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 50;
      padding: 16px;
    }
    .modal {
      background: #fff;
      border-radius: 12px;
      width: min(480px, 100%);
      padding: 20px;
      box-shadow: 0 20px 50px rgba(0, 0, 0, 0.2);
    }
    .modal-title {
      margin: 0 0 12px;
      font-size: 16px;
      font-weight: 700;
    }
    .modal .input, .modal select {
      width: 100%;
      height: 40px;
      border: 1px solid #d1d5db;
      border-radius: 8px;
      padding: 0 12px;
      font-size: 14px;
      background: #fff;
      color: var(--text);
    }
    .modal-actions {
      display: flex;
      gap: 10px;
      margin-top: 16px;
    }
    .token-warn {
      background: #fffbeb;
      border: 1px solid #fde68a;
      color: #92400e;
      border-radius: 8px;
      padding: 10px 12px;
      font-size: 13px;
      margin-bottom: 12px;
    }
    .token-plain {
      display: block;
      background: #f3f4f6;
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px;
      font-size: 13px;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      word-break: break-all;
      user-select: all;
    }
    @media (max-width: 768px) {
      body { padding: 12px; }
      .header, .auth, .pane { padding-left: 14px; padding-right: 14px; }
      .tabs { padding-left: 8px; padding-right: 8px; }
      th, td { font-size: 13px; padding: 10px 6px; }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1 class="title">管理后台</h1>
      <div id="identity" class="identity" hidden>
        <span id="identityText"></span>
        <button id="logoutBtn" class="btn-ghost btn-sm" type="button">退出</button>
      </div>
    </div>

    <!-- 登录区：单输入框，主密码与令牌皆可登录 -->
    <div id="auth" class="auth">
      <div class="muted">请输入管理密码或令牌登录（凭证保存在当前浏览器，无过期）。令牌登录仅管理自己名下的内容。</div>
      <div class="row">
        <input id="credential" type="password" placeholder="管理密码或令牌" autocomplete="off" />
        <button id="loginBtn" type="button">登录</button>
      </div>
      <div id="authError" class="error"></div>
    </div>

    <!-- 主区：分享 | 项目 | 令牌（令牌页签仅主密码身份显示） -->
    <div id="main" hidden>
      <div class="tabs">
        <button class="tab-btn active" data-tab="shares" type="button">分享</button>
        <button class="tab-btn" data-tab="projects" type="button">项目</button>
        <button class="tab-btn" data-tab="tokens" id="tabBtnTokens" type="button">令牌</button>
      </div>

      <div class="pane" id="pane-shares">
        <div class="feedback" id="sharesFeedback"></div>
        <div class="table-wrap" id="sharesTableWrap"></div>
      </div>

      <div class="pane" id="pane-projects" hidden>
        <div class="create-row">
          <input id="newProjectName" class="input" type="text" placeholder="新项目名称" />
          <button id="createProjectBtn" type="button">新建项目</button>
        </div>
        <div class="feedback" id="projectsFeedback"></div>
        <div class="table-wrap" id="projectsTableWrap"></div>
      </div>

      <div class="pane" id="pane-tokens" hidden>
        <div class="create-row">
          <input id="newTokenName" class="input" type="text" placeholder="令牌名称（如：巡检 Agent）" />
          <button id="createTokenBtn" type="button">签发令牌</button>
        </div>
        <div class="feedback" id="tokensFeedback"></div>
        <div class="table-wrap" id="tokensTableWrap"></div>
      </div>
    </div>
  </div>

  <!-- 通用弹层：挂载/移动选择器、令牌明文一次性展示 -->
  <div class="overlay" id="overlay" hidden>
    <div class="modal" role="dialog" aria-modal="true">
      <div id="modalContent"></div>
    </div>
  </div>

  <script>
    (function () {
      "use strict";

      // 凭证持久化 key：主密码或令牌明文（登录后所有请求以 Bearer 头携带）
      var CREDENTIAL_KEY = "wx-admin-credential";

      var authEl = document.getElementById("auth");
      var credentialEl = document.getElementById("credential");
      var loginBtnEl = document.getElementById("loginBtn");
      var authErrorEl = document.getElementById("authError");
      var mainEl = document.getElementById("main");
      var identityEl = document.getElementById("identity");
      var identityTextEl = document.getElementById("identityText");
      var logoutBtnEl = document.getElementById("logoutBtn");
      var tabBtnTokensEl = document.getElementById("tabBtnTokens");
      var overlayEl = document.getElementById("overlay");
      var modalContentEl = document.getElementById("modalContent");

      var paneEls = {
        shares: document.getElementById("pane-shares"),
        projects: document.getElementById("pane-projects"),
        tokens: document.getElementById("pane-tokens")
      };
      var feedbackEls = {
        shares: document.getElementById("sharesFeedback"),
        projects: document.getElementById("projectsFeedback"),
        tokens: document.getElementById("tokensFeedback")
      };
      var tableWrapEls = {
        shares: document.getElementById("sharesTableWrap"),
        projects: document.getElementById("projectsTableWrap"),
        tokens: document.getElementById("tokensTableWrap")
      };

      var credential = "";   // 当前凭证（主密码或令牌明文）
      var me = null;         // /api/auth/me 结果：{type, name, isOwner}
      var currentTab = "shares";

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
        var date = new Date(value);
        if (Number.isNaN(date.getTime())) return "-";
        return date.toLocaleString("zh-CN", { hour12: false });
      }

      // 统一请求封装：携带凭证头、解析 JSON、非 2xx 抛出 data.error；
      // 401 视为凭证失效，清状态回登录页
      async function apiFetch(path, options) {
        options = options || {};
        var headers = { "Authorization": "Bearer " + credential };
        if (options.body !== undefined) {
          headers["Content-Type"] = "application/json";
        }
        var response = await fetch(path, {
          method: options.method || "GET",
          headers: headers,
          body: options.body
        });
        var data = null;
        try {
          data = await response.json();
        } catch (err) {
          data = null;
        }
        if (response.status === 401) {
          handleExpired();
          throw new Error("凭证已失效");
        }
        if (!response.ok) {
          throw new Error((data && data.error) || ("请求失败（HTTP " + response.status + "）"));
        }
        return { status: response.status, data: data };
      }

      function handleExpired() {
        credential = "";
        me = null;
        localStorage.removeItem(CREDENTIAL_KEY);
        closeModal();
        mainEl.hidden = true;
        authEl.hidden = false;
        authErrorEl.textContent = "凭证已失效";
      }

      function setFeedback(tab, text, isError) {
        var el = feedbackEls[tab];
        el.textContent = text || "";
        el.className = isError ? "feedback error" : "feedback";
      }

      // ==================== 登录 / 身份 ====================

      async function login(cred, isAuto) {
        authErrorEl.textContent = "";
        loginBtnEl.disabled = true;
        loginBtnEl.textContent = "登录中...";

        try {
          var response = await fetch("/api/auth/me", {
            headers: { "Authorization": "Bearer " + cred }
          });
          if (!response.ok) {
            if (response.status === 401) {
              localStorage.removeItem(CREDENTIAL_KEY);
              throw new Error(isAuto ? "凭证已失效" : "密码或令牌错误");
            }
            var data = await response.json().catch(function () { return {}; });
            throw new Error((data && data.error) || "登录失败");
          }
          var profile = await response.json();
          credential = cred;
          me = profile;
          localStorage.setItem(CREDENTIAL_KEY, cred);
          enterMain();
        } catch (error) {
          authErrorEl.textContent = error && error.message ? error.message : "登录失败";
        } finally {
          loginBtnEl.disabled = false;
          loginBtnEl.textContent = "登录";
        }
      }

      function enterMain() {
        authErrorEl.textContent = "";
        authEl.hidden = true;
        identityEl.hidden = false;
        identityTextEl.innerHTML = escapeHTML(me.name) +
          " <span class='badge " + (me.isOwner ? "badge-owner" : "badge-token") + "'>" +
          (me.isOwner ? "主密码·全量" : "令牌·仅自己名下") + "</span>";
        // 令牌页签仅主密码（isOwner）身份可见
        tabBtnTokensEl.style.display = me.isOwner ? "" : "none";
        mainEl.hidden = false;
        currentTab = "shares";
        switchTab("shares");
      }

      function logout() {
        credential = "";
        me = null;
        localStorage.removeItem(CREDENTIAL_KEY);
        closeModal();
        mainEl.hidden = true;
        identityEl.hidden = true;
        credentialEl.value = "";
        authEl.hidden = false;
        authErrorEl.textContent = "";
        credentialEl.focus();
      }

      // ==================== 标签页 ====================

      function switchTab(tab) {
        if (tab === "tokens" && (!me || !me.isOwner)) {
          tab = "shares";
        }
        currentTab = tab;
        var btns = document.querySelectorAll(".tab-btn");
        for (var i = 0; i < btns.length; i++) {
          btns[i].classList.toggle("active", btns[i].getAttribute("data-tab") === tab);
        }
        Object.keys(paneEls).forEach(function (name) {
          paneEls[name].hidden = name !== tab;
        });
        if (tab === "shares") loadShares();
        if (tab === "projects") loadProjects();
        if (tab === "tokens") loadTokens();
      }

      // ==================== 分享页签 ====================

      async function loadShares() {
        setFeedback("shares", "正在加载分享列表...");
        try {
          var res = await apiFetch("/api/shares");
          var items = (res.data && res.data.items) || [];
          renderShares(items);
          setFeedback("shares", "共 " + items.length + " 条记录");
        } catch (error) {
          if (credential) setFeedback("shares", error.message, true);
          renderShares([]);
        }
      }

      function renderShares(items) {
        if (!items.length) {
          tableWrapEls.shares.innerHTML = "<div class='muted'>暂无分享记录</div>";
          return;
        }
        var rows = items.map(function (item) {
          var id = escapeHTML(item.id);
          var title = escapeHTML(item.title || "无标题");
          var style = escapeHTML(item.style || "-");
          var projectName = escapeHTML(item.projectName || "-");
          var creator = escapeHTML(item.creatorTokenName || "-");
          var createdAt = escapeHTML(formatDate(item.createdAt));
          var updatedAt = escapeHTML(formatDate(item.updatedAt));
          var detachBtn = item.projectId
            ? "<button class='btn-ghost btn-sm' data-action='detach' data-id='" + id + "'>移出项目</button>"
            : "";
          return "<tr>" +
            "<td class='nowrap'><a href='/s/" + id + "' target='_blank' rel='noopener noreferrer'>" + id + "</a></td>" +
            "<td>" + title + "</td>" +
            "<td class='nowrap'>" + style + "</td>" +
            "<td>" + projectName + "</td>" +
            "<td>" + creator + "</td>" +
            "<td class='nowrap'>" + createdAt + "</td>" +
            "<td class='nowrap'>" + updatedAt + "</td>" +
            "<td><div class='ops'>" +
            "<button class='btn-ghost btn-sm' data-action='attach' data-id='" + id +
            "' data-title='" + escapeHTML(item.title || item.id) + "' data-project-id='" + escapeHTML(item.projectId || "") + "'>挂载/移动</button>" +
            detachBtn +
            "<button class='btn-danger btn-sm' data-action='delete' data-id='" + id + "'>删除</button>" +
            "</div></td>" +
            "</tr>";
        }).join("");

        // 除标题列外都固定宽度，标题列吃掉剩余宽度，避免标题被挤成竖排
        tableWrapEls.shares.innerHTML = "<table class='shares-table'>" +
          "<colgroup>" +
          "<col style='width:88px;'>" +
          "<col>" +
          "<col style='width:120px;'>" +
          "<col style='width:132px;'>" +
          "<col style='width:100px;'>" +
          "<col style='width:160px;'>" +
          "<col style='width:160px;'>" +
          "<col style='width:190px;'>" +
          "</colgroup>" +
          "<thead><tr><th>ID</th><th>标题</th><th>样式</th><th>所属项目</th><th>创建者</th><th>创建时间</th><th>更新时间</th><th>操作</th></tr></thead>" +
          "<tbody>" + rows + "</tbody>" +
          "</table>";
      }

      async function deleteShare(id, buttonEl) {
        if (!window.confirm("确认删除该分享记录？删除后不可恢复。")) {
          return;
        }
        buttonEl.disabled = true;
        try {
          await apiFetch("/api/share/" + encodeURIComponent(id), { method: "DELETE" });
          await loadShares();
        } catch (error) {
          if (credential) setFeedback("shares", error.message, true);
          buttonEl.disabled = false;
        }
      }

      async function detachShare(id, projectIdAfter) {
        if (!window.confirm("移出后该文章变为独立分享，链接保持有效。确认移出？")) {
          return;
        }
        try {
          await apiFetch("/api/share/" + encodeURIComponent(id) + "/detach", { method: "POST" });
          if (projectIdAfter) {
            // 项目详情内移除：刷新项目列表并重新展开该项目的详情
            var existing = document.querySelector("[data-project-detail='" + projectIdAfter + "']");
            if (existing) existing.remove();
            await loadProjects();
            renderProjectDetail(projectIdAfter);
          } else {
            await loadShares();
          }
        } catch (error) {
          if (credential) setFeedback(currentTab === "projects" ? "projects" : "shares", error.message, true);
        }
      }

      // ==================== 挂载 / 移动弹层 ====================

      async function openAttachModal(shareId, shareTitle, currentProjectId, onDone) {
        // 下拉数据来自 GET /api/projects（服务端按身份过滤：令牌仅自己名下）
        var projects = [];
        try {
          var res = await apiFetch("/api/projects");
          projects = (res.data && res.data.items) || [];
        } catch (error) {
          if (!credential) return;
          setFeedback("shares", error.message, true);
          return;
        }

        var options = ["<option value='__new__'>➕ 新建项目…</option>"];
        projects.forEach(function (p) {
          var selected = currentProjectId && p.id === currentProjectId ? " selected" : "";
          options.push("<option value='" + escapeHTML(p.id) + "'" + selected + ">" +
            escapeHTML(p.name) + "（" + p.shareCount + " 篇）</option>");
        });

        showModal(
          "<h3 class='modal-title'>挂载 / 移动到项目</h3>" +
          "<div class='muted' style='margin-bottom:12px;'>分享：" + escapeHTML(shareTitle) + "</div>" +
          "<select id='attachSelect'>" + options.join("") + "</select>" +
          "<input id='attachNewName' class='input' type='text' placeholder='新项目名称' " +
          "style='margin-top:10px; display:none;' />" +
          "<div id='attachError' class='error' style='margin-top:8px;'></div>" +
          "<div class='modal-actions'>" +
          "<button id='attachConfirm' type='button'>确定</button>" +
          "<button id='attachCancel' class='btn-ghost' type='button'>取消</button>" +
          "</div>"
        );

        var selectEl = document.getElementById("attachSelect");
        var newNameEl = document.getElementById("attachNewName");
        var attachErrorEl = document.getElementById("attachError");

        // 下拉默认项就是「新建项目」时 change 不触发（分享尚未归属项目，
        // 或用户重新选中同一个选项），所以初始状态必须显式同步一次，
        // 否则界面停留在没有输入框的状态，确认只会报「请输入新项目名称」
        function syncNewNameField(userChanged) {
          var isNew = selectEl.value === "__new__";
          newNameEl.style.display = isNew ? "" : "none";
          attachErrorEl.textContent = "";
          if (isNew && userChanged) newNameEl.focus();
        }
        selectEl.addEventListener("change", function () { syncNewNameField(true); });
        syncNewNameField(false);

        document.getElementById("attachCancel").addEventListener("click", closeModal);
        document.getElementById("attachConfirm").addEventListener("click", async function () {
          // 已有项目按 projectId 精确挂载（跨创建者同名时不会解析错目标），
          // 新建分支按名字让服务端自动创建
          var payload = null;
          if (selectEl.value === "__new__") {
            var projectName = newNameEl.value.trim();
            if (!projectName) {
              attachErrorEl.textContent = "请输入新项目名称";
              newNameEl.focus();
              return;
            }
            payload = { project: projectName };
          } else {
            payload = { projectId: selectEl.value };
          }
          var confirmBtn = this;
          confirmBtn.disabled = true;
          try {
            await apiFetch("/api/share/" + encodeURIComponent(shareId) + "/attach", {
              method: "POST",
              body: JSON.stringify(payload)
            });
            closeModal();
            await loadShares();
            if (typeof onDone === "function") await onDone();
          } catch (error) {
            confirmBtn.disabled = false;
            if (credential) {
              attachErrorEl.textContent = error.message;
            }
          }
        });
      }

      // ==================== 项目页签 ====================

      async function loadProjects() {
        setFeedback("projects", "正在加载项目列表...");
        try {
          var res = await apiFetch("/api/projects");
          var items = (res.data && res.data.items) || [];
          renderProjects(items);
          setFeedback("projects", "共 " + items.length + " 个项目");
        } catch (error) {
          if (credential) setFeedback("projects", error.message, true);
          renderProjects([]);
        }
      }

      function renderProjects(items) {
        if (!items.length) {
          tableWrapEls.projects.innerHTML = "<div class='muted'>暂无项目</div>";
          return;
        }
        var rows = items.map(function (p) {
          var id = escapeHTML(p.id);
          var name = escapeHTML(p.name);
          var creator = escapeHTML(p.creatorTokenName || "站长");
          var lastShareAt = escapeHTML(formatDate(p.lastShareAt));
          var createdAt = escapeHTML(formatDate(p.createdAt));
          return "<tr data-project-row='" + id + "'>" +
            "<td>" + name + "</td>" +
            "<td style='width:70px;'>" + p.shareCount + "</td>" +
            "<td style='width:160px;'>" + lastShareAt + "</td>" +
            "<td style='width:110px;'>" + creator + "</td>" +
            "<td style='width:160px;'>" + createdAt + "</td>" +
            "<td style='width:200px;'><div class='ops'>" +
            "<button class='btn-ghost btn-sm' data-action='view' data-id='" + id + "'>查看</button>" +
            "<button class='btn-ghost btn-sm' data-action='rename' data-id='" + id + "' data-name='" + name + "'>重命名</button>" +
            "<button class='btn-danger btn-sm' data-action='delete-project' data-id='" + id + "' data-count='" + p.shareCount + "'>删除</button>" +
            "</div></td>" +
            "</tr>";
        }).join("");

        tableWrapEls.projects.innerHTML = "<table style='min-width:860px;'>" +
          "<thead><tr><th>名称</th><th>文章数</th><th>最新更新</th><th>创建者</th><th>创建时间</th><th>操作</th></tr></thead>" +
          "<tbody>" + rows + "</tbody>" +
          "</table>";
      }

      async function createProject() {
        var name = document.getElementById("newProjectName").value.trim();
        if (!name) {
          setFeedback("projects", "项目名不能为空", true);
          return;
        }
        var btn = document.getElementById("createProjectBtn");
        btn.disabled = true;
        try {
          var res = await apiFetch("/api/projects", {
            method: "POST",
            body: JSON.stringify({ name: name })
          });
          // 后端幂等语义：201 新建成功；200 表示范围内同名项目已存在，直接复用
          if (res.status === 201) {
            setFeedback("projects", "项目已创建");
          } else {
            setFeedback("projects", "该项目已存在");
          }
          document.getElementById("newProjectName").value = "";
          await loadProjects();
        } catch (error) {
          if (credential) setFeedback("projects", error.message === "同名项目已存在" ? "该项目已存在" : error.message, true);
        } finally {
          btn.disabled = false;
        }
      }

      function toggleProjectDetail(projectId, buttonEl) {
        var existing = document.querySelector("[data-project-detail='" + projectId + "']");
        if (existing) {
          existing.remove();
          if (buttonEl) buttonEl.textContent = "查看";
          return;
        }
        if (buttonEl) buttonEl.textContent = "收起";
        renderProjectDetail(projectId);
      }

      async function renderProjectDetail(projectId) {
        var row = document.querySelector("[data-project-row='" + projectId + "']");
        if (!row) return;
        var tr = document.createElement("tr");
        tr.setAttribute("data-project-detail", projectId);
        tr.innerHTML = "<td colspan='6' class='detail-cell'><div class='muted'>正在加载文章列表...</div></td>";
        row.after(tr);

        try {
          // 项目详情为公开接口，这里统一带凭证便于一致处理
          var res = await apiFetch("/api/projects/" + encodeURIComponent(projectId));
          var shares = (res.data && res.data.shares) || [];
          if (!shares.length) {
            tr.innerHTML = "<td colspan='6' class='detail-cell'><div class='muted'>该项目暂无文章</div></td>";
            return;
          }
          var listHTML = shares.map(function (s) {
            var id = escapeHTML(s.id);
            return "<div class='detail-item'>" +
              "<span class='detail-title'><a href='/s/" + id + "' target='_blank' rel='noopener noreferrer'>" +
              escapeHTML(s.title || "无标题") + "</a></span>" +
              "<span class='muted'>" + escapeHTML(formatDate(s.createdAt)) + "</span>" +
              "<button class='btn-ghost btn-sm' data-action='move-share' data-id='" + id +
              "' data-title='" + escapeHTML(s.title || "无标题") +
              "' data-project-id='" + escapeHTML(projectId) + "'>移动</button>" +
              "<button class='btn-ghost btn-sm' data-action='detach-share' data-id='" + id +
              "' data-project-id='" + escapeHTML(projectId) + "'>移除</button>" +
              "</div>";
          }).join("");
          tr.innerHTML = "<td colspan='6' class='detail-cell'>" + listHTML + "</td>";
        } catch (error) {
          if (!credential) {
            tr.remove();
            return;
          }
          tr.innerHTML = "<td colspan='6' class='detail-cell'><div class='error'>" + escapeHTML(error.message) + "</div></td>";
        }
      }

      async function renameProject(projectId, currentName) {
        var input = window.prompt("请输入新的项目名称：", currentName);
        if (input === null) return;
        var newName = input.trim();
        if (!newName) {
          setFeedback("projects", "项目名不能为空", true);
          return;
        }
        if (newName === currentName) return;
        if (!window.confirm("重命名后，仍使用旧名字发布的 Agent 将会自动创建一个新项目，文章会分流到新项目。确认重命名？")) {
          return;
        }
        try {
          await apiFetch("/api/projects/" + encodeURIComponent(projectId), {
            method: "PATCH",
            body: JSON.stringify({ name: newName })
          });
          setFeedback("projects", "项目已重命名");
          await loadProjects();
        } catch (error) {
          if (credential) setFeedback("projects", error.message, true);
        }
      }

      async function deleteProject(projectId, shareCount) {
        if (!window.confirm("项目下的 " + shareCount + " 篇文章将变为独立分享，链接保持有效。确认删除项目？")) {
          return;
        }
        try {
          await apiFetch("/api/projects/" + encodeURIComponent(projectId), { method: "DELETE" });
          setFeedback("projects", "项目已删除");
          await loadProjects();
        } catch (error) {
          if (credential) setFeedback("projects", error.message, true);
        }
      }

      // ==================== 令牌页签（仅主密码） ====================

      async function loadTokens() {
        setFeedback("tokens", "正在加载令牌列表...");
        try {
          var res = await apiFetch("/api/tokens");
          var items = (res.data && res.data.items) || [];
          renderTokens(items);
          setFeedback("tokens", "共 " + items.length + " 个令牌");
        } catch (error) {
          if (credential) setFeedback("tokens", error.message, true);
          renderTokens([]);
        }
      }

      function renderTokens(items) {
        if (!items.length) {
          tableWrapEls.tokens.innerHTML = "<div class='muted'>暂无令牌</div>";
          return;
        }
        var rows = items.map(function (t) {
          var id = escapeHTML(t.id);
          var name = escapeHTML(t.name);
          var createdAt = escapeHTML(formatDate(t.createdAt));
          var badge = t.revoked
            ? "<span class='badge badge-revoked'>已吊销</span>"
            : "<span class='badge badge-active'>有效</span>";
          var op = t.revoked
            ? "<button class='btn-danger btn-sm' disabled>吊销</button>"
            : "<button class='btn-danger btn-sm' data-action='revoke' data-id='" + id + "'>吊销</button>";
          return "<tr>" +
            "<td>" + name + "</td>" +
            "<td style='width:90px;'>" + badge + "</td>" +
            "<td style='width:170px;'>" + createdAt + "</td>" +
            "<td style='width:90px;'>" + op + "</td>" +
            "</tr>";
        }).join("");

        tableWrapEls.tokens.innerHTML = "<table style='min-width:560px;'>" +
          "<thead><tr><th>名称</th><th>状态</th><th>创建时间</th><th>操作</th></tr></thead>" +
          "<tbody>" + rows + "</tbody>" +
          "</table>";
      }

      async function createToken() {
        var name = document.getElementById("newTokenName").value.trim();
        if (!name) {
          setFeedback("tokens", "令牌名不能为空", true);
          return;
        }
        var btn = document.getElementById("createTokenBtn");
        btn.disabled = true;
        try {
          var res = await apiFetch("/api/tokens", {
            method: "POST",
            body: JSON.stringify({ name: name })
          });
          document.getElementById("newTokenName").value = "";
          showTokenModal(name, (res.data && res.data.token) || "");
          await loadTokens();
        } catch (error) {
          if (credential) setFeedback("tokens", error.message, true);
        } finally {
          btn.disabled = false;
        }
      }

      async function revokeToken(id) {
        if (!window.confirm("吊销后使用该令牌的 Agent 或浏览器将立即失效。确认吊销？")) {
          return;
        }
        try {
          await apiFetch("/api/tokens/" + encodeURIComponent(id), { method: "DELETE" });
          setFeedback("tokens", "令牌已吊销");
          await loadTokens();
        } catch (error) {
          if (credential) setFeedback("tokens", error.message, true);
        }
      }

      // ==================== 弹层 ====================

      function showModal(inner) {
        modalContentEl.innerHTML = inner;
        overlayEl.hidden = false;
      }

      function closeModal() {
        overlayEl.hidden = true;
        modalContentEl.innerHTML = "";
      }

      function showTokenModal(name, tokenPlain) {
        showModal(
          "<h3 class='modal-title'>令牌已签发</h3>" +
          "<div class='token-warn'>令牌仅此一次显示，请立即复制保存</div>" +
          "<div class='muted' style='margin-bottom:6px;'>名称：" + escapeHTML(name) + "</div>" +
          "<code class='token-plain' id='tokenPlain'>" + escapeHTML(tokenPlain) + "</code>" +
          "<div class='modal-actions'>" +
          "<button id='tokenCopyBtn' type='button'>复制令牌</button>" +
          "<button id='tokenCloseBtn' class='btn-ghost' type='button'>关闭</button>" +
          "</div>"
        );

        document.getElementById("tokenCloseBtn").addEventListener("click", closeModal);
        document.getElementById("tokenCopyBtn").addEventListener("click", async function () {
          var ok = await copyText(tokenPlain);
          var btn = this;
          btn.textContent = ok ? "已复制" : "复制失败，请手动选择复制";
          setTimeout(function () { btn.textContent = "复制令牌"; }, 2000);
        });
      }

      // 复制：navigator.clipboard 优先，execCommand 降级（与编辑器 copyShareUrl 同策略）
      async function copyText(text) {
        try {
          await navigator.clipboard.writeText(text);
          return true;
        } catch (err) {
          try {
            var textarea = document.createElement("textarea");
            textarea.value = text;
            document.body.appendChild(textarea);
            textarea.select();
            document.execCommand("copy");
            document.body.removeChild(textarea);
            return true;
          } catch (err2) {
            return false;
          }
        }
      }

      // ==================== 事件绑定 ====================

      loginBtnEl.addEventListener("click", function () {
        var cred = (credentialEl.value || "").trim();
        if (!cred) {
          authErrorEl.textContent = "请输入密码或令牌";
          return;
        }
        login(cred, false);
      });

      credentialEl.addEventListener("keydown", function (event) {
        if (event.key === "Enter") loginBtnEl.click();
      });

      logoutBtnEl.addEventListener("click", logout);

      document.querySelectorAll(".tab-btn").forEach(function (btn) {
        btn.addEventListener("click", function () {
          switchTab(btn.getAttribute("data-tab"));
        });
      });

      document.getElementById("createProjectBtn").addEventListener("click", createProject);
      document.getElementById("newProjectName").addEventListener("keydown", function (event) {
        if (event.key === "Enter") createProject();
      });

      document.getElementById("createTokenBtn").addEventListener("click", createToken);
      document.getElementById("newTokenName").addEventListener("keydown", function (event) {
        if (event.key === "Enter") createToken();
      });

      // 分享表：操作事件委托
      tableWrapEls.shares.addEventListener("click", function (event) {
        var target = event.target;
        if (!(target instanceof HTMLButtonElement)) return;
        var action = target.getAttribute("data-action");
        var id = target.getAttribute("data-id");
        if (!action || !id) return;
        if (action === "delete") {
          deleteShare(id, target);
        } else if (action === "detach") {
          detachShare(id, "");
        } else if (action === "attach") {
          openAttachModal(
            id,
            target.getAttribute("data-title") || id,
            target.getAttribute("data-project-id") || ""
          );
        }
      });

      // 项目表：操作事件委托（含详情行内的移除按钮）
      tableWrapEls.projects.addEventListener("click", function (event) {
        var target = event.target;
        if (!(target instanceof HTMLButtonElement)) return;
        var action = target.getAttribute("data-action");
        var id = target.getAttribute("data-id");
        if (!action || !id) return;
        if (action === "view") {
          toggleProjectDetail(id, target);
        } else if (action === "rename") {
          renameProject(id, target.getAttribute("data-name") || "");
        } else if (action === "delete-project") {
          deleteProject(id, target.getAttribute("data-count") || "0");
        } else if (action === "detach-share") {
          detachShare(id, target.getAttribute("data-project-id") || "");
        } else if (action === "move-share") {
          var projectId = target.getAttribute("data-project-id") || "";
          openAttachModal(
            id,
            target.getAttribute("data-title") || id,
            projectId,
            async function () {
              await loadProjects();
              renderProjectDetail(projectId);
            }
          );
        }
      });

      // 令牌表：操作事件委托
      tableWrapEls.tokens.addEventListener("click", function (event) {
        var target = event.target;
        if (!(target instanceof HTMLButtonElement)) return;
        if (target.disabled) return;
        var action = target.getAttribute("data-action");
        var id = target.getAttribute("data-id");
        if (action === "revoke" && id) revokeToken(id);
      });

      // 刷新自动登录
      var cached = "";
      try {
        cached = localStorage.getItem(CREDENTIAL_KEY) || "";
      } catch (err) {
        cached = "";
      }
      if (cached) {
        login(cached, true);
      }
    })();
  </script>
</body>
</html>`

	return pageTemplate
}
