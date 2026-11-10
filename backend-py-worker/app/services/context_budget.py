"""Keep model prompts inside a character budget.

The worker used to paste the whole extracted document into every prompt. A
long thesis then either overflowed the model context or hit the 180 s gateway
timeout. This module picks the passages that matter for a question, and splits
long documents for map-reduce summarization.

Term extraction and chunk sizes mirror backend-go/internal/textsearch so that
local ranking (document scope) and database retrieval (project scope) behave
the same way.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

CHUNK_SIZE = 800
CHUNK_STEP = 700
MAX_TERMS = 24

_STOP_BIGRAMS = {
    "什么", "怎么", "如何", "哪些", "哪个", "这个", "那个", "一下", "是否", "可以", "我们", "你们",
    "他们", "请问", "有没", "没有", "是不", "不是", "一个", "这份", "文档", "帮我", "告诉", "总结",
    "内容", "主要", "情况", "为什", "的是", "了吗", "多少", "有哪", "是什",
}
_HAN = re.compile(r"[一-鿿]+")
_ASCII = re.compile(r"[A-Za-z0-9]+")
_TOKEN = re.compile(r"[一-鿿]+|[A-Za-z0-9]+")


@dataclass(slots=True)
class Passage:
    label: str
    content: str
    document_id: str = ""
    title: str = ""
    version_no: int = 0
    chunk_no: int = 0

    def as_context(self) -> dict[str, object]:
        return {
            "label": self.label,
            "document_id": self.document_id,
            "title": self.title,
            "version_no": self.version_no,
            "content": self.content,
        }

    def as_citation(self) -> dict[str, object]:
        snippet = self.content.strip().replace("\n", " ")
        return {
            "label": self.label,
            "document_id": self.document_id,
            "title": self.title,
            "version_no": self.version_no,
            "chunk_no": self.chunk_no,
            "snippet": snippet[:120] + ("…" if len(snippet) > 120 else ""),
        }


def chunk(text: str) -> list[str]:
    body = text.strip()
    if not body:
        return []
    out: list[str] = []
    start = 0
    while start < len(body):
        end = min(start + CHUNK_SIZE, len(body))
        out.append(body[start:end])
        if end == len(body):
            break
        start += CHUNK_STEP
    return out


def terms(query: str) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []

    def add(term: str) -> None:
        if term and term not in seen and term not in _STOP_BIGRAMS and len(out) < MAX_TERMS:
            seen.add(term)
            out.append(term)

    for token in _TOKEN.findall(query):
        if _ASCII.fullmatch(token):
            if len(token) >= 2:
                add(token.lower())
        elif len(token) == 2:
            add(token)
        elif len(token) > 2:
            for i in range(len(token) - 1):
                add(token[i : i + 2])
    return out


def score(content: str, query_terms: list[str]) -> int:
    lowered = content.lower()
    return sum(1 for t in query_terms if t in lowered)


def select_passages(
    text: str,
    question: str,
    budget_chars: int,
    *,
    document_id: str = "",
    title: str = "",
    version_no: int = 0,
) -> list[Passage]:
    """Return labelled passages of one document that fit the budget.

    Short documents are passed whole. Long ones are chunked and the chunks
    matching the most question terms are kept, in document order so the
    model reads them as a coherent excerpt. When nothing matches (e.g. "总结
    这份文档"), the opening chunks are used.
    """
    body = text.strip()
    if not body:
        return []
    meta = {"document_id": document_id, "title": title, "version_no": version_no}
    if len(body) <= budget_chars:
        return [Passage(label="片段1", content=body, chunk_no=0, **meta)]
    chunks = chunk(body)
    query_terms = terms(question)
    ranked = sorted(range(len(chunks)), key=lambda i: (-score(chunks[i], query_terms), i))
    if not query_terms or score(chunks[ranked[0]], query_terms) == 0:
        ranked = list(range(len(chunks)))
    chosen: list[int] = []
    used = 0
    for i in ranked:
        if used + len(chunks[i]) > budget_chars:
            continue
        chosen.append(i)
        used += len(chunks[i])
    if not chosen:
        # Budget smaller than one chunk: keep the most relevant chunk, cut to
        # the budget, rather than sending no text at all.
        best = ranked[0]
        return [Passage(label="片段1", content=chunks[best][:budget_chars], chunk_no=best, **meta)]
    chosen.sort()
    return [
        Passage(label=f"片段{n + 1}", content=chunks[i], chunk_no=i, **meta)
        for n, i in enumerate(chosen)
    ]


def passages_from_hits(hits: list[dict[str, object]], budget_chars: int) -> list[Passage]:
    """Turn backend search hits (best first) into labelled passages."""
    out: list[Passage] = []
    used = 0
    for hit in hits:
        content = hit.get("content")
        if not isinstance(content, str) or not content.strip():
            continue
        if used + len(content) > budget_chars:
            continue
        used += len(content)
        version_no = hit.get("version_no")
        chunk_no = hit.get("chunk_no")
        out.append(
            Passage(
                label=f"片段{len(out) + 1}",
                content=content,
                document_id=str(hit.get("document_id") or ""),
                title=str(hit.get("title") or ""),
                version_no=version_no if isinstance(version_no, int) else 0,
                chunk_no=chunk_no if isinstance(chunk_no, int) else 0,
            )
        )
    return out


def split_for_summary(text: str, section_chars: int, max_sections: int) -> tuple[list[str], bool]:
    """Split a long text into at most max_sections pieces of ~section_chars.

    Returns (sections, sampled). When the document needs more sections than
    allowed, sections are sampled evenly across it and sampled is True, so the
    summary can say it did not read every page.
    """
    body = text.strip()
    if not body:
        return [], False
    pieces = [body[i : i + section_chars] for i in range(0, len(body), section_chars)]
    if len(pieces) <= max_sections:
        return pieces, False
    step = (len(pieces) - 1) / (max_sections - 1) if max_sections > 1 else 0
    picked = sorted({round(i * step) for i in range(max_sections)})
    return [pieces[i] for i in picked], True
