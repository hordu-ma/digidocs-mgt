from __future__ import annotations

import csv
import io
import json
import os
import re
import shutil
import subprocess
import tempfile
import xml.etree.ElementTree as ET
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
    if ext == ".xlsx":
        return _extract_xlsx(content)
    if ext == ".pptx":
        return _extract_pptx(content)
    if ext == ".doc":
        return _extract_doc_binary(content)
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


_MAX_ZIP_ENTRY = 50 * 1024 * 1024  # 50 MB per entry guard


def _safe_zip_read(archive: zipfile.ZipFile, name: str) -> bytes:
    info = archive.getinfo(name)
    if info.file_size > _MAX_ZIP_ENTRY:
        raise DocumentTextExtractionError(f"ZIP 内文件 {name} 过大（{info.file_size} 字节），跳过抽取")
    return archive.read(name)


def _extract_xlsx(content: bytes) -> str:
    """Extract cell text from .xlsx by parsing sheet XMLs with shared-string resolution."""
    ns = {"s": "http://schemas.openxmlformats.org/spreadsheetml/2006/main"}
    try:
        with zipfile.ZipFile(io.BytesIO(content)) as archive:
            # Build shared strings table
            shared: list[str] = []
            if "xl/sharedStrings.xml" in archive.namelist():
                ss_xml = _safe_zip_read(archive, "xl/sharedStrings.xml")
                ss_root = ET.fromstring(ss_xml)
                for si in ss_root.findall("s:si", ns):
                    parts: list[str] = []
                    for t_el in si.iter("{http://schemas.openxmlformats.org/spreadsheetml/2006/main}t"):
                        if t_el.text:
                            parts.append(t_el.text)
                    shared.append("".join(parts))

            # Parse each sheet in order
            sheet_names = sorted(
                n for n in archive.namelist()
                if re.match(r"xl/worksheets/sheet\d+\.xml$", n)
            )
            all_rows: list[str] = []
            for sheet_name in sheet_names:
                sheet_xml = _safe_zip_read(archive, sheet_name)
                sheet_root = ET.fromstring(sheet_xml)
                for row_el in sheet_root.findall(".//s:row", ns):
                    cells: list[str] = []
                    for c_el in row_el.findall("s:c", ns):
                        cell_type = c_el.get("t", "")
                        v_el = c_el.find("s:v", ns)
                        is_el = c_el.find("s:is", ns)
                        cell_text = ""
                        if cell_type == "s" and v_el is not None and v_el.text:
                            idx = int(v_el.text)
                            if 0 <= idx < len(shared):
                                cell_text = shared[idx]
                        elif cell_type == "inlineStr" and is_el is not None:
                            parts = []
                            for t_el in is_el.iter(
                                "{http://schemas.openxmlformats.org/spreadsheetml/2006/main}t"
                            ):
                                if t_el.text:
                                    parts.append(t_el.text)
                            cell_text = "".join(parts)
                        elif v_el is not None and v_el.text:
                            cell_text = v_el.text
                        cells.append(cell_text.strip())
                    row_text = " | ".join(cells)
                    if row_text.replace("|", "").strip():
                        all_rows.append(row_text)
    except DocumentTextExtractionError:
        raise
    except Exception as exc:
        raise DocumentTextExtractionError("xlsx 文件结构不完整，无法抽取正文") from exc

    text = "\n".join(all_rows).strip()
    if not text:
        raise DocumentTextExtractionError("xlsx 文件中未提取到可用正文")
    return text


def _extract_pptx(content: bytes) -> str:
    """Extract visible text from .pptx slides, preserving paragraph boundaries."""
    a_ns = "http://schemas.openxmlformats.org/drawingml/2006/main"
    try:
        with zipfile.ZipFile(io.BytesIO(content)) as archive:
            slide_names = sorted(
                n for n in archive.namelist()
                if re.match(r"ppt/slides/slide\d+\.xml$", n)
            )
            slides_text: list[str] = []
            for slide_name in slide_names:
                slide_xml = _safe_zip_read(archive, slide_name)
                root = ET.fromstring(slide_xml)
                paragraphs: list[str] = []
                for p_el in root.iter(f"{{{a_ns}}}p"):
                    runs: list[str] = []
                    for child in p_el:
                        tag = child.tag.split("}")[-1] if "}" in child.tag else child.tag
                        if tag == "r":
                            for t_el in child.iter(f"{{{a_ns}}}t"):
                                if t_el.text:
                                    runs.append(t_el.text)
                        elif tag == "br":
                            runs.append("\n")
                    para_text = "".join(runs).strip()
                    if para_text:
                        paragraphs.append(para_text)
                if paragraphs:
                    slides_text.append("\n".join(paragraphs))
    except DocumentTextExtractionError:
        raise
    except Exception as exc:
        raise DocumentTextExtractionError("pptx 文件结构不完整，无法抽取正文") from exc

    text = "\n\n".join(slides_text).strip()
    if not text:
        raise DocumentTextExtractionError("pptx 文件中未提取到可用正文")
    return text


def _extract_doc_binary(content: bytes) -> str:
    """Extract text from old .doc binary format using antiword."""
    antiword = shutil.which("antiword")
    if antiword is None:
        raise DocumentTextExtractionError("当前环境未安装 antiword，无法抽取 .doc 正文")

    with tempfile.NamedTemporaryFile(prefix="digidocs-doc-", suffix=".doc", delete=False) as tmp:
        tmp.write(content)
        tmp_path = tmp.name
    try:
        result = subprocess.run(
            [antiword, "-m", "UTF-8.txt", tmp_path],
            check=True,
            capture_output=True,
            text=True,
            timeout=60,
        )
    except subprocess.TimeoutExpired as exc:
        raise DocumentTextExtractionError("antiword 执行超时") from exc
    except subprocess.CalledProcessError as exc:
        raise DocumentTextExtractionError(
            f"antiword 执行失败: {exc.stderr.strip() or exc}"
        ) from exc
    finally:
        try:
            os.remove(tmp_path)
        except OSError:
            pass

    text = result.stdout.strip()
    if not text:
        raise DocumentTextExtractionError(".doc 文件中未提取到可用正文")
    return text


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
