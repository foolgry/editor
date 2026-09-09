#!/usr/bin/env python3
"""wxmd-publish - 公众号 Markdown 编辑器线上发布工具（纯 HTTP，零依赖）。

子命令：
  publish   发布 Markdown 内容，返回分享链接（本地图片自动上传并改写引用）
  get       获取分享内容
  list      列出全部分享（需要 WXMD_LIST_PASSWORD）
  delete    删除分享（需要 WXMD_LIST_PASSWORD）

环境变量：
  WXMD_API_URL        API 地址，默认 https://md.foolgry.top
  WXMD_API_TIMEOUT    请求超时秒数，默认 30
  WXMD_LIST_PASSWORD  列表/删除操作的管理密码
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

STYLES = {
    "wechat-default": "默认公众号风格",
    "latepost-depth": "晚点风格",
    "wechat-ft": "金融时报",
    "wechat-anthropic": "Claude",
    "wechat-claude-song": "Claude Song",
    "wechat-tech": "技术风格",
    "wechat-elegant": "优雅简约",
    "wechat-deepread": "深度阅读",
    "wechat-jonyive": "Jony Ive",
    "wechat-apple": "Apple 极简",
    "kenya-emptiness": "原研哉·空",
    "hische-editorial": "Hische·编辑部",
    "ando-concrete": "安藤·清水",
    "gaudi-organic": "高迪·有机",
    "kami": "Kami",
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

    result = http_json(f"{base}/api/share", method="POST",
                       payload={"content": content, "style": args.style})
    share_id = result.get("id")
    output = {
        "id": share_id,
        "url": f"{base}/s/{share_id}",
        "style": args.style,
        "uploadedImages": uploaded,
    }
    if warnings:
        output["warnings"] = warnings
    print(json.dumps(output, ensure_ascii=False, indent=2))

    if args.open and share_id:
        webbrowser.open(output["url"])


def require_password():
    password = os.environ.get("WXMD_LIST_PASSWORD", "").strip()
    if not password:
        raise PublishError("需要管理密码：请设置环境变量 WXMD_LIST_PASSWORD")
    return {"X-List-Password": password}


def cmd_get(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/share/{args.id}"), ensure_ascii=False, indent=2))


def cmd_list(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/shares", headers=require_password()),
                     ensure_ascii=False, indent=2))


def cmd_delete(args):
    base = base_url(args)
    print(json.dumps(http_json(f"{base}/api/share/{args.id}", method="DELETE",
                               headers=require_password()), ensure_ascii=False, indent=2))


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
    p_pub.add_argument("--style", default="wechat-default",
                       help=f"排版样式，默认 wechat-default，可选: {', '.join(STYLES)}")
    p_pub.add_argument("--open", action="store_true", help="发布后用浏览器打开分享链接")
    p_pub.set_defaults(func=cmd_publish)

    p_get = sub.add_parser("get", help="获取分享内容")
    p_get.add_argument("id", help="分享 ID")
    p_get.set_defaults(func=cmd_get)

    p_list = sub.add_parser("list", help="列出全部分享（需要 WXMD_LIST_PASSWORD）")
    p_list.set_defaults(func=cmd_list)

    p_del = sub.add_parser("delete", help="删除分享（需要 WXMD_LIST_PASSWORD）")
    p_del.add_argument("id", help="分享 ID")
    p_del.set_defaults(func=cmd_delete)

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
