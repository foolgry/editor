#!/usr/bin/env python3
"""wxmd-publish - 公众号 Markdown 编辑器线上发布工具（纯 HTTP，零依赖）。

子命令：
  publish        发布 Markdown 内容，返回分享链接（图片自动上传并改写引用）
  get            获取分享内容
  list           列出全部分享（需要 WXMD_LIST_PASSWORD 或 WXMD_TOKEN）
  delete         删除分享（需要 WXMD_LIST_PASSWORD 或 WXMD_TOKEN）
  projects       列出自己名下的项目（需要 WXMD_TOKEN）
  project-create 创建项目（需要 WXMD_TOKEN）
  project-rename 重命名项目（需要 WXMD_TOKEN）
  attach         把分享挂载到项目（需要 WXMD_TOKEN）
  detach         把分享移出项目（需要 WXMD_TOKEN）
  set-token      把令牌写入技能目录下的 .env（需要 WXMD_TOKEN 的操作一次配置即可）
  token-status   显示当前生效的凭证来源与掩码，排查"令牌没生效"

本站已关闭匿名发布：publish 及其图片上传都必须带令牌，未设置 WXMD_TOKEN 会直接
报错（主密码 WXMD_LIST_PASSWORD 可作为站长凭证回退）。还没有令牌时到
<API 地址>/apply 申请。

凭证与配置来自两处，环境变量优先，其次技能目录下的 .env（默认无需任何环境变量）：

  <技能目录>/.env     形如 WXMD_TOKEN=wmt_xxxx，可手动编辑，也可用
                      `publish.py set-token` 写入（会同时把文件权限设为 600）

可配置项：
  WXMD_API_URL        API 地址，默认 https://md.foolgry.top
  WXMD_API_TIMEOUT    请求超时秒数，默认 30
  WXMD_TOKEN          发布/项目相关操作的令牌，优先于 WXMD_LIST_PASSWORD
  WXMD_LIST_PASSWORD  列表/删除的管理密码；发布与项目相关操作未设 WXMD_TOKEN 时
                      回退用它（主密码视作站长凭证）
"""

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
import uuid
import webbrowser
from pathlib import Path

DEFAULT_BASE_URL = "https://md.foolgry.top"
DEFAULT_TIMEOUT = 30
DEFAULT_STYLE = "kami-slides"

STYLES = {
    "wechat-default": "默认公众号风格",
    "latepost-depth": "晚点风格",
    "wechat-ft": "金融时报",
    "wechat-anthropic": "Claude",
    "wechat-claude-song": "Claude Song",
    "wechat-tech": "技术风格",
    "wechat-elegant": "优雅简约",
    "wechat-deepread": "深度阅读",
    "wechat-nyt": "纽约时报",
    "wechat-jonyive": "Jony Ive",
    "wechat-medium": "Medium 长文",
    "wechat-apple": "Apple 极简",
    "kenya-emptiness": "原研哉·空",
    "hische-editorial": "Hische·编辑部",
    "ando-concrete": "安藤·清水",
    "gaudi-organic": "高迪·有机",
    "kami-resume": "Kami · 简历",
    "kami-print": "Kami · 白底单页",
    "kami-report": "Kami · 财报研报",
    "kami-slides": "Kami · 演讲文稿",
    "kami-product": "Kami · 产品简报",
    "kami-letter": "Kami · 正式信函",
    "kami-changelog": "Kami · 更新日志",
    "kami-portfolio": "Kami · 沉静画册",
    "guardian": "Guardian 卫报",
    "nikkei": "Nikkei 日経",
    "lemonde": "Le Monde 世界报",
}

IMAGE_EXTS = {".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"}
MIME_BY_EXT = {
    ".png": "image/png",
    ".jpg": "image/jpeg",
    ".jpeg": "image/jpeg",
    ".gif": "image/gif",
    ".webp": "image/webp",
    ".svg": "image/svg+xml",
}

# Markdown 图片：![alt](path) 或 ![alt](<path>)，可带 "title"
MD_IMAGE_RE = re.compile(r"!\[[^\]]*\]\(\s*(<[^>]+>|[^)\s]+)(?:\s+\"[^\"]*\")?\s*\)")
# HTML 图片：<img src="path">
HTML_IMAGE_RE = re.compile(r"(<img\b[^>]*?\bsrc=[\"'])([^\"']+)([\"'])", re.IGNORECASE)


class PublishError(Exception):
    pass


# 技能目录 = scripts/ 的上一级；.env 与 SKILL.md 同级，便于用户直接找到并编辑
SKILL_ROOT = Path(__file__).resolve().parent.parent
ENV_FILE = SKILL_ROOT / ".env"
SCRIPT_PATH = Path(__file__).resolve()

ENV_TEMPLATE = """# wxmd-publish 凭证与配置
# 本文件已被 .gitignore 忽略，令牌不会进仓库；也可用 `publish.py set-token` 写入。
# 检查当前生效的凭证：python3 scripts/publish.py token-status

# 发布令牌（wmt_ 开头）。publish（含图片上传）与全部项目操作都需要
# 申请入口见 SKILL.md，或访问 <API 地址>/apply
"""


def _unquote_env_value(raw):
    """解析 .env 的值：去包裹引号并识别行尾注释。

    单引号内原样保留；双引号内按 shell 习惯解析 \\n \\t \\" \\\\；引号之外
    的 ` #` 起视为行尾注释（令牌不含空格，这个粒度够用）。
    """
    raw = raw.strip()
    if not raw:
        return ""
    if raw[0] in ("'", '"'):
        quote = raw[0]
        out = []
        i = 1
        while i < len(raw):
            ch = raw[i]
            if ch == "\\" and quote == '"' and i + 1 < len(raw):
                nxt = raw[i + 1]
                out.append({"n": "\n", "t": "\t", '"': '"', "\\": "\\"}.get(nxt, "\\" + nxt))
                i += 2
                continue
            if ch == quote:
                break  # 引号闭合，后面的内容（通常是注释）忽略
            out.append(ch)
            i += 1
        return "".join(out)
    for i, ch in enumerate(raw):
        if ch == "#" and i > 0 and raw[i - 1].isspace():
            return raw[:i].rstrip()
    return raw


def parse_env_file(path):
    """解析 .env：逐行 KEY=VALUE，支持 export 前缀、引号与 # 注释。

    文件缺失或不可读时返回空字典——.env 是可选便利，不该让脚本直接失败。
    """
    result = {}
    try:
        text = path.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError):
        return result
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line[len("export "):].lstrip()
        key, sep, value = line.partition("=")
        key = key.strip()
        if not sep or not key:
            continue
        result[key] = _unquote_env_value(value.strip())
    return result


_ENV_CACHE = None


def _load_env():
    global _ENV_CACHE
    if _ENV_CACHE is None:
        _ENV_CACHE = parse_env_file(ENV_FILE)
    return _ENV_CACHE


def env_value(name):
    """取配置：真实环境变量优先，其次技能目录下的 .env；空值视作未设置。"""
    from_env = (os.environ.get(name) or "").strip()
    if from_env:
        return from_env
    return (_load_env().get(name) or "").strip()


def env_source(name):
    """说明当前生效值来自哪里：'环境变量' / '.env' / ''（未设置）。

    排查"令牌明明写进 .env 了却没生效"时，靠它区分是不是被环境变量顶掉了。
    """
    if (os.environ.get(name) or "").strip():
        return "环境变量"
    if (_load_env().get(name) or "").strip():
        return ".env"
    return ""


def mask_secret(value):
    """只露头尾，够辨认是哪个令牌，又不足以被拿去用。"""
    if len(value) <= 12:
        return f"****（共 {len(value)} 字符）"
    return f"{value[:8]}…{value[-4:]}（共 {len(value)} 字符）"


def ensure_env_file():
    """.env 不存在时用模板创建，让用户/agent 有个现成文件可改。"""
    if not ENV_FILE.is_file():
        ENV_FILE.parent.mkdir(parents=True, exist_ok=True)
        ENV_FILE.write_text(ENV_TEMPLATE, encoding="utf-8")
    try:
        ENV_FILE.chmod(0o600)
    except OSError:
        pass
    return ENV_FILE


def write_env_value(key, value):
    """把 KEY=VALUE 写进 .env：已有该键则原地替换，否则追加，其余内容保持不变。"""
    ensure_env_file()
    lines = ENV_FILE.read_text(encoding="utf-8").splitlines()
    for i, old in enumerate(lines):
        stripped = old.strip()
        body = stripped[len("export "):].lstrip() if stripped.startswith("export ") else stripped
        if body.partition("=")[0].strip() == key:
            lines[i] = f"{key}={value}"
            break
    else:
        if lines and lines[-1].strip():
            lines.append("")
        lines.append(f"{key}={value}")
    ENV_FILE.write_text("\n".join(lines) + "\n", encoding="utf-8")
    ENV_FILE.chmod(0o600)
    global _ENV_CACHE
    _ENV_CACHE = None
    return ENV_FILE


def script_hint():
    """报错时给出的可复制命令，用绝对路径，避免用户猜脚本在哪。"""
    return f"python3 {SCRIPT_PATH}"


def base_url(args):
    return (args.base_url or env_value("WXMD_API_URL") or DEFAULT_BASE_URL).rstrip("/")


def timeout():
    raw = env_value("WXMD_API_TIMEOUT")
    try:
        return max(1, int(raw))
    except ValueError:
        return DEFAULT_TIMEOUT


def absolute_url(base, path):
    """把服务端返回的相对路径（如 /p/<pid>）拼成完整 URL；空值原样返回。"""
    if not path:
        return path
    if path.startswith(("http://", "https://")):
        return path
    return f"{base}/{path.lstrip('/')}"


def origin_of(url):
    """从 API URL 反推服务地址，用于把服务端返回的 applyUrl 拼成完整链接。"""
    marker = "/api/"
    if marker in url:
        return url.split(marker, 1)[0]
    return url.rstrip("/")


def format_http_error(code, body_text, url):
    """把服务端错误体整理成提示。凭证类失败会带 applyUrl，拼成完整申请链接，
    否则使用者只会看到一句“未授权”，不知道该去哪里拿令牌。"""
    data = {}
    try:
        parsed = json.loads(body_text)
        if isinstance(parsed, dict):
            data = parsed
    except ValueError:
        pass

    msg = data.get("error") or body_text
    apply_path = data.get("applyUrl")
    if apply_path:
        msg = f"{msg}（申请令牌：{origin_of(url)}{apply_path}）"
    return f"HTTP {code}: {msg}"


def http_json(url, method="GET", payload=None, headers=None):
    data = None
    req_headers = {"Accept": "application/json"}
    if headers:
        req_headers.update(headers)
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        req_headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, method=method, headers=req_headers)
    try:
        with urllib.request.urlopen(req, timeout=timeout()) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        raise PublishError(format_http_error(
            e.code, e.read().decode("utf-8", errors="replace"), url))
    except urllib.error.URLError as e:
        raise PublishError(f"无法连接服务器: {e.reason}")


def upload_image(base, image_path, auth_headers_map=None):
    boundary = f"----wxmd{uuid.uuid4().hex}"
    ext = image_path.suffix.lower()
    mime = MIME_BY_EXT.get(ext, "application/octet-stream")
    file_bytes = image_path.read_bytes()

    body = b"\r\n".join([
        f"--{boundary}".encode(),
        f'Content-Disposition: form-data; name="file"; filename="{image_path.name}"'.encode(),
        f"Content-Type: {mime}".encode(),
        b"",
        file_bytes,
        f"--{boundary}--".encode(),
        b"",
    ])

    req_headers = {"Content-Type": f"multipart/form-data; boundary={boundary}"}
    if auth_headers_map:
        req_headers.update(auth_headers_map)

    url = f"{base}/api/upload"
    req = urllib.request.Request(
        url,
        data=body,
        method="POST",
        headers=req_headers,
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout()) as resp:
            result = json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        raise PublishError(format_http_error(
            e.code, e.read().decode("utf-8", errors="replace"), url))
    except urllib.error.URLError as e:
        raise PublishError(f"无法连接服务器: {e.reason}")

    url = result.get("url")
    if not url:
        raise PublishError("服务器未返回图片 URL")
    return url


def is_remote_ref(ref):
    return ref.startswith(("http://", "https://", "//", "data:", "img://"))


def resolve_local_ref(ref, base_dir):
    """把 Markdown 中的图片引用解析为本地文件路径，不存在则返回 None。"""
    cleaned = ref.strip()
    if cleaned.startswith("<") and cleaned.endswith(">"):
        cleaned = cleaned[1:-1]
    cleaned = urllib.parse.unquote(cleaned)
    path = Path(cleaned)
    if not path.is_absolute():
        path = base_dir / path
    path = path.resolve()
    if path.is_file() and path.suffix.lower() in IMAGE_EXTS:
        return path
    return None


def process_images(content, base_dir, base, warnings, auth):
    """上传内容中引用的本地图片并把引用改写为线上 URL，返回 (新内容, 上传数)。

    auth 为发布凭证请求头：上传接口与发布同口径，未带有效令牌会被拒。
    """
    cache = {}
    uploaded = 0

    def upload_and_replace(ref):
        nonlocal uploaded
        if is_remote_ref(ref):
            return None
        path = resolve_local_ref(ref, base_dir)
        if path is None:
            warnings.append(f"本地图片不存在或格式不支持，已保留原引用: {ref}")
            return None
        key = str(path)
        if key not in cache:
            try:
                cache[key] = base + upload_image(base, path, auth)
                uploaded += 1
            except PublishError as e:
                warnings.append(f"图片上传失败，已保留原引用: {ref} ({e})")
                cache[key] = None
        return cache[key]

    def replace_md(match):
        ref = match.group(1)
        new_url = upload_and_replace(ref)
        if new_url is None:
            return match.group(0)
        return match.group(0).replace(ref, new_url, 1)

    def replace_html(match):
        prefix, ref, suffix = match.groups()
        new_url = upload_and_replace(ref)
        if new_url is None:
            return match.group(0)
        return prefix + new_url + suffix

    content = MD_IMAGE_RE.sub(replace_md, content)
    content = HTML_IMAGE_RE.sub(replace_html, content)
    return content, uploaded


def read_content(args):
    if args.text is not None:
        return args.text, Path.cwd()
    if args.file:
        path = Path(args.file).expanduser().resolve()
        if not path.is_file():
            raise PublishError(f"文件不存在: {args.file}")
        return path.read_text(encoding="utf-8"), path.parent
    if not sys.stdin.isatty():
        return sys.stdin.read(), Path.cwd()
    raise PublishError("缺少内容：请使用 --file、--text 或 stdin 管道输入")


def cmd_publish(args):
    base = base_url(args)
    content, base_dir = read_content(args)
    if not content.strip():
        raise PublishError("内容不能为空")

    # 本站已关闭匿名发布：先在发布前取好凭证，图片上传和发布都要用它。
    # 缺少令牌时这里直接失败，不会上传一半再报错。
    auth = auth_headers(require_token=True, base=base)

    warnings = []
    content, uploaded = process_images(content, base_dir, base, warnings, auth)

    payload = {"content": content, "style": args.style}
    if args.project:
        payload["project"] = args.project

    result = http_json(f"{base}/api/share", method="POST",
                       payload=payload, headers=auth)
    share_id = result.get("id")
    output = {
        "id": share_id,
        "url": f"{base}/s/{share_id}",
        "style": args.style,
        "uploadedImages": uploaded,
        "projectId": result.get("projectId"),
        "projectUrl": absolute_url(base, result.get("projectUrl")),
        "deepUrl": absolute_url(base, result.get("deepUrl")),
    }
    if warnings:
        output["warnings"] = warnings
    print(json.dumps(output, ensure_ascii=False, indent=2))

    if args.open and share_id:
        webbrowser.open(output["url"])


def auth_headers(require_token=False, base=None):
    """构造认证请求头，优先 WXMD_TOKEN（Bearer），其次 WXMD_LIST_PASSWORD。

    require_token=True：发布（含图片上传）与项目相关操作（publish --project、
    projects、project-create、project-rename、attach、detach），主密码视作站长
    凭证，同样以 Bearer 发送。
    require_token=False：list/delete，WXMD_TOKEN 以 Bearer 发送；
    WXMD_LIST_PASSWORD 沿用 X-List-Password 头（维持现状）。

    取值顺序为「环境变量 > .env」（见 env_value）。base 有值时，缺少凭证的报错
    会附上令牌申请入口。
    """
    apply_hint = f"；申请令牌：{base}/apply" if base else ""
    token = env_value("WXMD_TOKEN")
    if token:
        return {"Authorization": f"Bearer {token}"}
    password = env_value("WXMD_LIST_PASSWORD")
    if password:
        if require_token:
            return {"Authorization": f"Bearer {password}"}
        return {"X-List-Password": password}
    if require_token:
        raise PublishError(
            f"需要令牌：请在 {ENV_FILE} 里设置 WXMD_TOKEN=wmt_xxxx，"
            f"或执行 {script_hint()} set-token wmt_xxxx{apply_hint}")
    raise PublishError(
        f"需要管理密码：请在 {ENV_FILE} 里设置 WXMD_LIST_PASSWORD=xxxx，"
        f"或设置同名环境变量{apply_hint}")


def cmd_get(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/share/{args.id}"), ensure_ascii=False, indent=2))


def cmd_list(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/shares", headers=auth_headers(base=base)),
                     ensure_ascii=False, indent=2))


def cmd_delete(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/share/{args.id}", method="DELETE",
                               headers=auth_headers(base=base)), ensure_ascii=False, indent=2))


def cmd_projects(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/projects",
                               headers=auth_headers(require_token=True, base=base)),
                     ensure_ascii=False, indent=2))


def cmd_project_create(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/projects", method="POST",
                               payload={"name": args.name},
                               headers=auth_headers(require_token=True, base=base)),
                     ensure_ascii=False, indent=2))


def cmd_project_rename(args):
    base = base_url(args)
    headers = auth_headers(require_token=True, base=base)
    data = http_json(f"{base}/api/projects", headers=headers)
    items = data.get("items", []) if isinstance(data, dict) else data
    # 第一个参数允许是项目 ID（同名项目消歧出口）或项目名
    by_id = [p for p in items if isinstance(p, dict) and p.get("id") == args.old]
    if by_id:
        matches = by_id
        args.old = by_id[0].get("name")
    else:
        matches = [p for p in items if isinstance(p, dict) and p.get("name") == args.old]
    if not matches:
        names = ", ".join(str(p.get("name")) for p in items
                          if isinstance(p, dict) and p.get("name")) or "（无）"
        raise PublishError(f"项目不存在: {args.old}。可用项目: {names}")
    if len(matches) > 1:
        # 主密码凭证可见全部项目，不同创建者可能同名：列出候选要求用 ID 消歧
        candidates = ", ".join(
            f"{p.get('id')}（创建者: {p.get('creatorTokenName') or '站长'}）"
            for p in matches)
        raise PublishError(f"存在多个同名项目 {args.old}，请改用项目 ID 消歧: {candidates}")
    project = matches[0]
    http_json(f"{base}/api/projects/{project.get('id')}", method="PATCH",
              payload={"name": args.new}, headers=headers)
    print(json.dumps({"id": project.get("id"), "oldName": args.old, "newName": args.new},
                     ensure_ascii=False, indent=2))


def print_share(result, base):
    """输出 share 对象，projectUrl/deepUrl 相对路径拼成完整 URL。"""
    if isinstance(result, dict):
        result = dict(result)
        result["projectUrl"] = absolute_url(base, result.get("projectUrl"))
        result["deepUrl"] = absolute_url(base, result.get("deepUrl"))
    print(json.dumps(result, ensure_ascii=False, indent=2))


def cmd_attach(args):
    base = base_url(args)
    if not args.project and not args.project_id:
        raise PublishError("请指定 --project（项目名）或 --project-id（项目 ID）")
    # --project-id 精确挂载（跨创建者同名项目的消歧出口）；默认按名字自动创建
    payload = {"projectId": args.project_id} if args.project_id else {"project": args.project}
    result = http_json(f"{base}/api/share/{args.id}/attach", method="POST",
                       payload=payload,
                       headers=auth_headers(require_token=True, base=base))
    print_share(result, base)


def cmd_detach(args):
    base = base_url(args)
    result = http_json(f"{base}/api/share/{args.id}/detach", method="POST",
                       headers=auth_headers(require_token=True, base=base))
    print_share(result, base)


def cmd_set_token(args):
    """把令牌写进技能目录下的 .env，用户和 agent 都不用去找文件、也不用手改格式。

    令牌优先取位置参数；未给时从 stdin 读（管道传入不会留在 shell 历史里）。
    """
    if args.token is not None:
        raw = args.token
    elif not sys.stdin.isatty():
        raw = sys.stdin.read()
    else:
        raw = ""
    token = raw.strip()
    if not token:
        raise PublishError(
            "缺少令牌："
            f"{script_hint()} set-token wmt_xxxx，"
            "或 echo 'wmt_xxxx' | "
            f"{script_hint()} set-token"
        )
    write_env_value("WXMD_TOKEN", token)
    result = {
        "ok": True,
        "envFile": str(ENV_FILE),
        "token": mask_secret(token),
        "hint": "已写入，之后发布无需再传令牌；可用 token-status 确认",
    }
    if not token.startswith("wmt_"):
        result["warning"] = "令牌通常以 wmt_ 开头，请确认没有写错或漏掉字符"
    print(json.dumps(result, ensure_ascii=False, indent=2))


def cmd_token_status(args):
    """报告当前生效的凭证来源，用于确认 .env 有没有被读到。"""
    base = base_url(args)
    token = env_value("WXMD_TOKEN")
    password = env_value("WXMD_LIST_PASSWORD")
    print(json.dumps({
        "envFile": str(ENV_FILE),
        "envFileExists": ENV_FILE.is_file(),
        "apiUrl": base,
        "applyUrl": f"{base}/apply",
        "publishToken": {
            "configured": bool(token),
            "source": env_source("WXMD_TOKEN") or None,
            "value": mask_secret(token) if token else None,
        },
        "listPassword": {
            "configured": bool(password),
            "source": env_source("WXMD_LIST_PASSWORD") or None,
        },
        "ready": bool(token or password),
        "hint": "source 为 '环境变量' 时说明环境变量覆盖了 .env" if token else
                f"未配置令牌：在 {ENV_FILE} 里写 WXMD_TOKEN=wmt_xxxx，"
                f"或执行 {script_hint()} set-token wmt_xxxx",
    }, ensure_ascii=False, indent=2))


def cmd_styles(args):
    print(json.dumps([{"key": k, "name": v} for k, v in STYLES.items()],
                     ensure_ascii=False, indent=2))


def main():
    parser = argparse.ArgumentParser(
        prog="publish.py",
        description="公众号 Markdown 编辑器线上发布工具（纯 HTTP，零依赖）",
    )
    parser.add_argument("--base-url", help=f"API 地址，默认 {DEFAULT_BASE_URL}")
    sub = parser.add_subparsers(dest="command")

    p_pub = sub.add_parser("publish", help="发布 Markdown 内容，返回分享链接（需要 WXMD_TOKEN）")
    p_pub.add_argument("--file", help="本地 Markdown 文件路径（本地图片自动上传，同样需要令牌）")
    p_pub.add_argument("--text", help="直接传入 Markdown 文本")
    p_pub.add_argument("--style", default=DEFAULT_STYLE,
                       help=f"排版样式，默认 {DEFAULT_STYLE}，可选: {', '.join(STYLES)}")
    p_pub.add_argument("--project",
                       help="归入项目（按名字引用，不存在自动创建）")
    p_pub.add_argument("--open", action="store_true", help="发布后用浏览器打开分享链接")
    p_pub.set_defaults(func=cmd_publish)

    p_get = sub.add_parser("get", help="获取分享内容")
    p_get.add_argument("id", help="分享 ID")
    p_get.set_defaults(func=cmd_get)

    p_list = sub.add_parser("list", help="列出全部分享（需要 WXMD_LIST_PASSWORD 或 WXMD_TOKEN）")
    p_list.set_defaults(func=cmd_list)

    p_del = sub.add_parser("delete", help="删除分享（需要 WXMD_LIST_PASSWORD 或 WXMD_TOKEN）")
    p_del.add_argument("id", help="分享 ID")
    p_del.set_defaults(func=cmd_delete)

    p_projects = sub.add_parser("projects", help="列出自己名下的项目（需要 WXMD_TOKEN）")
    p_projects.set_defaults(func=cmd_projects)

    p_pcreate = sub.add_parser("project-create", help="创建项目（需要 WXMD_TOKEN）")
    p_pcreate.add_argument("name", help="项目名")
    p_pcreate.set_defaults(func=cmd_project_create)

    p_prename = sub.add_parser("project-rename", help="重命名项目（需要 WXMD_TOKEN）")
    p_prename.add_argument("old", help="旧项目名")
    p_prename.add_argument("new", help="新项目名")
    p_prename.set_defaults(func=cmd_project_rename)

    p_attach = sub.add_parser("attach", help="把分享挂载到项目（需要 WXMD_TOKEN）")
    p_attach.add_argument("id", help="分享 ID")
    p_attach.add_argument("--project",
                          help="目标项目名（不存在自动创建）")
    p_attach.add_argument("--project-id",
                          help="目标项目 ID（跨创建者同名项目的精确挂载，优先于 --project）")
    p_attach.set_defaults(func=cmd_attach)

    p_detach = sub.add_parser("detach", help="把分享移出项目（需要 WXMD_TOKEN）")
    p_detach.add_argument("id", help="分享 ID")
    p_detach.set_defaults(func=cmd_detach)

    p_styles = sub.add_parser("styles", help="列出可用排版样式")
    p_styles.set_defaults(func=cmd_styles)

    p_set = sub.add_parser("set-token", help="把令牌写入技能目录下的 .env")
    p_set.add_argument("token", nargs="?", help="令牌；省略则从 stdin 读取")
    p_set.set_defaults(func=cmd_set_token)

    p_status = sub.add_parser("token-status", help="显示当前生效的凭证来源（掩码）")
    p_status.set_defaults(func=cmd_token_status)

    args = parser.parse_args()
    if not getattr(args, "func", None):
        parser.print_help()
        sys.exit(2)

    try:
        args.func(args)
    except PublishError as e:
        print(json.dumps({"error": str(e)}, ensure_ascii=False), file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
