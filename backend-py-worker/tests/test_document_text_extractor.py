from __future__ import annotations

import subprocess
from pathlib import Path

import pytest

from app.services.document_text_extractor import (
    DocumentTextExtractionError,
    extract_text,
)


def test_extract_text_pdf_uses_pdftotext(monkeypatch) -> None:
    def fake_which(name: str) -> str | None:
        if name == "pdftotext":
            return "/usr/bin/pdftotext"
        return None

    def fake_run(args: list[str], check: bool, capture_output: bool, text: bool):
        assert args[0] == "/usr/bin/pdftotext"
        Path(args[-1]).write_text("PDF 正文", encoding="utf-8")
        return subprocess.CompletedProcess(args, 0, "", "")

    monkeypatch.setattr("app.services.document_text_extractor.shutil.which", fake_which)
    monkeypatch.setattr("app.services.document_text_extractor.subprocess.run", fake_run)

    extracted = extract_text("report.pdf", b"%PDF-1.4 fake")

    assert extracted == "PDF 正文"


def test_extract_text_image_uses_tesseract(monkeypatch) -> None:
    def fake_which(name: str) -> str | None:
        if name == "tesseract":
            return "/usr/bin/tesseract"
        return None

    def fake_run(args: list[str], check: bool, capture_output: bool, text: bool):
        assert args[0] == "/usr/bin/tesseract"
        return subprocess.CompletedProcess(args, 0, "图片识别结果\n", "")

    monkeypatch.setattr("app.services.document_text_extractor.shutil.which", fake_which)
    monkeypatch.setattr("app.services.document_text_extractor.subprocess.run", fake_run)

    extracted = extract_text("scan.png", b"fake-image")

    assert extracted == "图片识别结果"


def test_extract_text_scanned_pdf_requires_ocr_dependencies(monkeypatch) -> None:
    def fake_which(name: str) -> str | None:
        if name == "pdftotext":
            return "/usr/bin/pdftotext"
        return None

    def fake_run(args: list[str], check: bool, capture_output: bool, text: bool):
        Path(args[-1]).write_text("", encoding="utf-8")
        return subprocess.CompletedProcess(args, 0, "", "")

    monkeypatch.setattr("app.services.document_text_extractor.shutil.which", fake_which)
    monkeypatch.setattr("app.services.document_text_extractor.subprocess.run", fake_run)

    with pytest.raises(DocumentTextExtractionError) as exc_info:
        extract_text("scan.pdf", b"%PDF-1.4 scanned")

    assert "tesseract" in str(exc_info.value)


# ── xlsx tests ──────────────────────────────────────────────


def _make_xlsx_bytes(shared_strings: list[str], sheet_rows: list[list[tuple[str, str]]]) -> bytes:
    """Build a minimal .xlsx in memory.

    shared_strings: list of shared string values
    sheet_rows: list of rows, each row is list of (type, value) tuples
        type "s" → shared string index, "n" → number, "inlineStr" → inline
    """
    import io as _io
    import zipfile as _zf

    buf = _io.BytesIO()
    ns = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"

    with _zf.ZipFile(buf, "w") as zf:
        # sharedStrings.xml
        if shared_strings:
            ss_parts = [f'<?xml version="1.0" encoding="UTF-8"?>'
                        f'<sst xmlns="{ns}" count="{len(shared_strings)}">']
            for s in shared_strings:
                ss_parts.append(f"<si><t>{s}</t></si>")
            ss_parts.append("</sst>")
            zf.writestr("xl/sharedStrings.xml", "".join(ss_parts))

        # sheet1.xml
        rows_xml: list[str] = []
        for r_idx, row in enumerate(sheet_rows, 1):
            cells_xml: list[str] = []
            for c_idx, (ctype, cval) in enumerate(row):
                ref = f"A{r_idx}"  # simplified
                if ctype == "inlineStr":
                    cells_xml.append(f'<c r="{ref}" t="inlineStr"><is><t>{cval}</t></is></c>')
                elif ctype == "s":
                    cells_xml.append(f'<c r="{ref}" t="s"><v>{cval}</v></c>')
                else:
                    cells_xml.append(f'<c r="{ref}"><v>{cval}</v></c>')
            rows_xml.append(f'<row r="{r_idx}">{"".join(cells_xml)}</row>')

        sheet_xml = (f'<?xml version="1.0" encoding="UTF-8"?>'
                     f'<worksheet xmlns="{ns}"><sheetData>{"".join(rows_xml)}</sheetData></worksheet>')
        zf.writestr("xl/worksheets/sheet1.xml", sheet_xml)

    return buf.getvalue()


def test_extract_xlsx_shared_strings() -> None:
    content = _make_xlsx_bytes(
        shared_strings=["姓名", "成绩", "张三", "95"],
        sheet_rows=[
            [("s", "0"), ("s", "1")],
            [("s", "2"), ("s", "3")],
        ],
    )
    text = extract_text("data.xlsx", content)
    assert "姓名" in text
    assert "张三" in text
    assert "95" in text


def test_extract_xlsx_inline_strings() -> None:
    content = _make_xlsx_bytes(
        shared_strings=[],
        sheet_rows=[[("inlineStr", "内联文本")]],
    )
    text = extract_text("inline.xlsx", content)
    assert "内联文本" in text


def test_extract_xlsx_empty_raises() -> None:
    content = _make_xlsx_bytes(shared_strings=[], sheet_rows=[])
    with pytest.raises(DocumentTextExtractionError):
        extract_text("empty.xlsx", content)


# ── pptx tests ──────────────────────────────────────────────


def _make_pptx_bytes(slides: list[list[str]]) -> bytes:
    """Build a minimal .pptx in memory. slides: list of slides, each slide is list of paragraphs."""
    import io as _io
    import zipfile as _zf

    buf = _io.BytesIO()
    a_ns = "http://schemas.openxmlformats.org/drawingml/2006/main"
    p_ns = "http://schemas.openxmlformats.org/presentationml/2006/main"
    r_ns = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"

    with _zf.ZipFile(buf, "w") as zf:
        for i, paragraphs in enumerate(slides, 1):
            paras_xml: list[str] = []
            for para in paragraphs:
                paras_xml.append(f'<a:p xmlns:a="{a_ns}"><a:r><a:t>{para}</a:t></a:r></a:p>')
            slide_xml = (f'<?xml version="1.0" encoding="UTF-8"?>'
                         f'<p:sld xmlns:p="{p_ns}" xmlns:a="{a_ns}" xmlns:r="{r_ns}">'
                         f'<p:cSld><p:spTree>{"".join(paras_xml)}</p:spTree></p:cSld></p:sld>')
            zf.writestr(f"ppt/slides/slide{i}.xml", slide_xml)

    return buf.getvalue()


def test_extract_pptx_basic() -> None:
    content = _make_pptx_bytes([["第一页标题", "第一页内容"], ["第二页标题"]])
    text = extract_text("demo.pptx", content)
    assert "第一页标题" in text
    assert "第一页内容" in text
    assert "第二页标题" in text


def test_extract_pptx_empty_raises() -> None:
    content = _make_pptx_bytes([])
    with pytest.raises(DocumentTextExtractionError):
        extract_text("empty.pptx", content)


# ── doc tests ───────────────────────────────────────────────


def test_extract_doc_binary_uses_antiword(monkeypatch) -> None:
    def fake_which(name: str) -> str | None:
        if name == "antiword":
            return "/usr/bin/antiword"
        return None

    def fake_run(args, check, capture_output, text, timeout=None):
        assert args[0] == "/usr/bin/antiword"
        assert "-m" in args and "UTF-8.txt" in args
        return subprocess.CompletedProcess(args, 0, "文档正文内容\n", "")

    monkeypatch.setattr("app.services.document_text_extractor.shutil.which", fake_which)
    monkeypatch.setattr("app.services.document_text_extractor.subprocess.run", fake_run)

    extracted = extract_text("old.doc", b"\xd0\xcf\x11\xe0 fake doc")
    assert extracted == "文档正文内容"


def test_extract_doc_binary_missing_antiword(monkeypatch) -> None:
    monkeypatch.setattr(
        "app.services.document_text_extractor.shutil.which", lambda name: None
    )
    with pytest.raises(DocumentTextExtractionError, match="antiword"):
        extract_text("old.doc", b"\xd0\xcf\x11\xe0 fake doc")
