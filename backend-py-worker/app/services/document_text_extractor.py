from __future__ import annotations

import csv
import io
import json
import os
import re
import shutil
import subprocess
import tempfile
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
    if ext == ".pdf":
        return _extract_pdf(content)
    if ext in {".png", ".jpg", ".jpeg", ".bmp", ".tif", ".tiff", ".webp"}:
        return _extract_image_with_ocr(ext, content)
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


def _extract_pdf(content: bytes) -> str:
    pdftotext = shutil.which("pdftotext")
    if pdftotext is None:
        raise DocumentTextExtractionError("当前环境未安装 pdftotext，无法抽取 PDF 正文")

    with tempfile.TemporaryDirectory(prefix="digidocs-pdf-") as tmp_dir:
        pdf_path = os.path.join(tmp_dir, "source.pdf")
        txt_path = os.path.join(tmp_dir, "source.txt")
        with open(pdf_path, "wb") as file_obj:
            file_obj.write(content)

        try:
            subprocess.run(
                [pdftotext, "-layout", "-nopgbrk", pdf_path, txt_path],
                check=True,
                capture_output=True,
                text=True,
            )
        except subprocess.CalledProcessError as exc:
            raise DocumentTextExtractionError(
                f"pdftotext 执行失败: {exc.stderr.strip() or exc}"
            ) from exc

        extracted = ""
        if os.path.exists(txt_path):
            extracted = open(txt_path, "r", encoding="utf-8", errors="ignore").read().strip()
        if extracted:
            return extracted

        if shutil.which("tesseract") is None or shutil.which("pdftoppm") is None:
            raise DocumentTextExtractionError(
                "PDF 未提取到文本；若为扫描件，请在 Worker 主机安装 pdftoppm 与 tesseract 后重试"
            )
        return _extract_scanned_pdf(pdf_path, tmp_dir)


def _extract_scanned_pdf(pdf_path: str, tmp_dir: str) -> str:
    pdftoppm = shutil.which("pdftoppm")
    if pdftoppm is None:
        raise DocumentTextExtractionError("当前环境未安装 pdftoppm，无法抽取扫描 PDF")

    image_prefix = os.path.join(tmp_dir, "page")
    try:
        subprocess.run(
            [pdftoppm, "-png", pdf_path, image_prefix],
            check=True,
            capture_output=True,
            text=True,
        )
    except subprocess.CalledProcessError as exc:
        raise DocumentTextExtractionError(
            f"pdftoppm 执行失败: {exc.stderr.strip() or exc}"
        ) from exc

    pages = sorted(
        os.path.join(tmp_dir, file_name)
        for file_name in os.listdir(tmp_dir)
        if file_name.startswith("page-") and file_name.endswith(".png")
    )
    if not pages:
        raise DocumentTextExtractionError("扫描 PDF 未生成可 OCR 的页面图像")

    chunks = [_run_tesseract(page) for page in pages]
    text = "\n\n".join(chunk for chunk in chunks if chunk).strip()
    if text == "":
        raise DocumentTextExtractionError("扫描 PDF OCR 后未提取到可用正文")
    return text


def _extract_image_with_ocr(ext: str, content: bytes) -> str:
    with tempfile.NamedTemporaryFile(
        prefix="digidocs-image-", suffix=ext, delete=False
    ) as file_obj:
        file_obj.write(content)
        image_path = file_obj.name
    try:
        text = _run_tesseract(image_path)
    finally:
        try:
            os.remove(image_path)
        except OSError:
            pass
    if text == "":
        raise DocumentTextExtractionError("图片 OCR 后未提取到可用正文")
    return text


def _run_tesseract(file_path: str) -> str:
    tesseract = shutil.which("tesseract")
    if tesseract is None:
        raise DocumentTextExtractionError("当前环境未安装 tesseract，无法进行图片 OCR")

    try:
        completed = subprocess.run(
            [tesseract, file_path, "stdout", "-l", "chi_sim+eng"],
            check=True,
            capture_output=True,
            text=True,
        )
    except subprocess.CalledProcessError as exc:
        raise DocumentTextExtractionError(
            f"tesseract 执行失败: {exc.stderr.strip() or exc}"
        ) from exc
    return completed.stdout.strip()
