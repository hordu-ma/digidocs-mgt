from __future__ import annotations

import csv
import io
import json
import os
import re
import zipfile
from typing import cast


class DocumentTextExtractionError(RuntimeError):
    """Raised when a document cannot be converted into plain text."""


def extract_text(file_name: str, content: bytes) -> str:
    ext = os.path.splitext(file_name)[1].lower()
    if ext in {".txt", ".md", ".csv", ".json"}:
        return _extract_text_like(ext, content)
    if ext == ".docx":
        return _extract_docx(content)
    raise DocumentTextExtractionError(f"暂不支持从 {ext or '未知类型'} 文件中抽取正文")


def _extract_text_like(ext: str, content: bytes) -> str:
    text = content.decode("utf-8", errors="ignore").strip()
    if ext == ".csv":
        rows: list[str] = []
        reader = csv.reader(io.StringIO(text))
        for row in reader:
            rows.append(" | ".join(str(cell).strip() for cell in row))
        return "\n".join(rows).strip()
    if ext == ".json":
        try:
            return json.dumps(cast(object, json.loads(text)), ensure_ascii=False, indent=2)
        except json.JSONDecodeError:
            return text
    return text


def _extract_docx(content: bytes) -> str:
    try:
        with zipfile.ZipFile(io.BytesIO(content)) as archive:
            xml = archive.read("word/document.xml").decode("utf-8", errors="ignore")
    except Exception as exc:  # pragma: no cover - exact stdlib exceptions vary
        raise DocumentTextExtractionError("docx 文件结构不完整，无法抽取正文") from exc

    text = re.sub(r"</w:p>", "\n", xml)
    text = re.sub(r"<[^>]+>", "", text)
    text = re.sub(r"\n{2,}", "\n", text)
    cleaned = text.strip()
    if cleaned == "":
        raise DocumentTextExtractionError("docx 文件中未提取到可用正文")
    return cleaned
