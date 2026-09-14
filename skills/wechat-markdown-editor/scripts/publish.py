#!/usr/bin/env python3
"""wxmd-publish - 公众号 Markdown 编辑器线上发布工具（纯 HTTP，零依赖）。

子命令：
  publish        发布 Markdown 内容，返回分享链接（本地图片自动上传并改写引用）
  get            获取分享内容
  list           列出全部分享（需要 WXMD_LIST_PASSWORD 或 WXMD_TOKEN）
  delete         删除分享（需要 WXMD_LIST_PASSWORD 或 WXMD_TOKEN）
  projects       列出自己名下的项目（需要 WXMD_TOKEN）
  project-create 创建项目（需要 WXMD_TOKEN）
  project-rename 重命名项目（需要 WXMD_TOKEN）
  attach         把分享挂载到项目（需要 WXMD_TOKEN）
  detach         把分享移出项目（需要 WXMD_TOKEN）

环境变量：
  WXMD_API_URL        API 地址，默认 https://md.foolgry.top
  WXMD_API_TIMEOUT    请求超时秒数，默认 30
  WXMD_TOKEN          项目相关操作的令牌，优先于 WXMD_LIST_PASSWORD
  WXMD_LIST_PASSWORD  列表/删除的管理密码；项目相关操作未设 WXMD_TOKEN 时
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


def base_url(args):
    return (args.base_url or os.environ.get("WXMD_API_URL") or DEFAULT_BASE_URL).rstrip("/")


def timeout():
    raw = os.environ.get("WXMD_API_TIMEOUT", "")
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
        body = e.read().decode("utf-8", errors="replace")
        try:
            msg = json.loads(body).get("error", body)
        except ValueError:
            msg = body
        raise PublishError(f"HTTP {e.code}: {msg}")
    except urllib.error.URLError as e:
        raise PublishError(f"无法连接服务器: {e.reason}")


def upload_image(base, image_path):
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

    req = urllib.request.Request(
        f"{base}/api/upload",
        data=body,
        method="POST",
        headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout()) as resp:
            result = json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body_text = e.read().decode("utf-8", errors="replace")
        try:
            msg = json.loads(body_text).get("error", body_text)
        except ValueError:
            msg = body_text
        raise PublishError(f"HTTP {e.code}: {msg}")
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


def process_images(content, base_dir, base, warnings):
    """上传内容中引用的本地图片并把引用改写为线上 URL，返回 (新内容, 上传数)。"""
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
                cache[key] = base + upload_image(base, path)
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

    warnings = []
    content, uploaded = process_images(content, base_dir, base, warnings)

    payload = {"content": content, "style": args.style}
    headers = None
    if args.project:
        payload["project"] = args.project
        headers = auth_headers(require_token=True)

    result = http_json(f"{base}/api/share", method="POST",
                       payload=payload, headers=headers)
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


def auth_headers(require_token=False):
    """构造认证请求头，优先 WXMD_TOKEN（Bearer），其次 WXMD_LIST_PASSWORD。

    require_token=True：项目相关操作（publish --project、projects、project-create、
    project-rename、attach、detach），主密码视作站长凭证，同样以 Bearer 发送。
    require_token=False：list/delete，WXMD_TOKEN 以 Bearer 发送；
    WXMD_LIST_PASSWORD 沿用 X-List-Password 头（维持现状）。
    """
    token = os.environ.get("WXMD_TOKEN", "").strip()
    if token:
        return {"Authorization": f"Bearer {token}"}
    password = os.environ.get("WXMD_LIST_PASSWORD", "").strip()
    if password:
        if require_token:
            return {"Authorization": f"Bearer {password}"}
        return {"X-List-Password": password}
    if require_token:
        raise PublishError("需要令牌：请设置环境变量 WXMD_TOKEN")
    raise PublishError("需要管理密码：请设置环境变量 WXMD_LIST_PASSWORD")


def cmd_get(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/share/{args.id}"), ensure_ascii=False, indent=2))


def cmd_list(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/shares", headers=auth_headers()),
                     ensure_ascii=False, indent=2))


def cmd_delete(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/share/{args.id}", method="DELETE",
                               headers=auth_headers()), ensure_ascii=False, indent=2))


def cmd_projects(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/projects",
                               headers=auth_headers(require_token=True)),
                     ensure_ascii=False, indent=2))


def cmd_project_create(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/projects", method="POST",
                               payload={"name": args.name},
                               headers=auth_headers(require_token=True)),
                     ensure_ascii=False, indent=2))


def cmd_project_rename(args):
    base = base_url(args)
    headers = auth_headers(require_token=True)
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
                       headers=auth_headers(require_token=True))
    print_share(result, base)


def cmd_detach(args):
    base = base_url(args)
    result = http_json(f"{base}/api/share/{args.id}/detach", method="POST",
                       headers=auth_headers(require_token=True))
    print_share(result, base)


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

    p_pub = sub.add_parser("publish", help="发布 Markdown 内容，返回分享链接")
    p_pub.add_argument("--file", help="本地 Markdown 文件路径（本地图片会自动上传）")
    p_pub.add_argument("--text", help="直接传入 Markdown 文本")
    p_pub.add_argument("--style", default=DEFAULT_STYLE,
                       help=f"排版样式，默认 {DEFAULT_STYLE}，可选: {', '.join(STYLES)}")
    p_pub.add_argument("--project",
                       help="归入项目（按名字引用，不存在自动创建；需要 WXMD_TOKEN）")
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
