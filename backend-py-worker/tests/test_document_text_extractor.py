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
