// Package textsearch holds the lexical retrieval rules shared by AI question
// answering and the user-facing full-text search: how extracted document text
// is chunked, and how a query is turned into match terms.
//
// Retrieval is lexical on purpose. The OpenClaw gateway exposes no working
// embeddings endpoint (POST /v1/embeddings returns 500 as of 2026-11-10) and
// the PostgreSQL image ships without pgvector, so dense retrieval would need
// new infrastructure. Character bigrams give usable recall for Chinese, which
// has no word boundaries, without a segmenter dependency. The chunk table is
// the extension point for embeddings later.
package textsearch

import (
	"strings"
	"unicode"
)

const (
	// ChunkSize / ChunkStep must match migration 014's backfill.
	ChunkSize = 800
	ChunkStep = 700
	// MaxChunks bounds one version's footprint (~280k characters).
	MaxChunks = 400
	// MaxTerms bounds query fan-out.
	MaxTerms = 24
)

// Chunk splits text into overlapping rune windows.
func Chunk(text string) []string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}
	chunks := make([]string, 0, len(runes)/ChunkStep+1)
	for start := 0; start < len(runes) && len(chunks) < MaxChunks; start += ChunkStep {
		end := start + ChunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
	}
	return chunks
}

// stopBigrams are question scaffolding that would match almost any chunk.
var stopBigrams = map[string]bool{
	"什么": true, "怎么": true, "如何": true, "哪些": true, "哪个": true, "这个": true, "那个": true,
	"一下": true, "是否": true, "可以": true, "我们": true, "你们": true, "他们": true, "请问": true,
	"有没": true, "没有": true, "是不": true, "不是": true, "一个": true, "这份": true, "文档": true,
	"帮我": true, "告诉": true, "总结": true, "内容": true, "主要": true, "情况": true, "为什": true,
	"的是": true, "了吗": true, "多少": true, "有哪": true, "是什": true,
}

// Terms extracts match terms from a query: lowercase ASCII words of 2+
// characters, and CJK runs as bigrams. A two-character run is kept whole; a
// lone CJK character is dropped (mostly particles such as 吗、的、了).
func Terms(query string) []string {
	seen := map[string]bool{}
	terms := make([]string, 0, MaxTerms)
	add := func(t string) {
		if t == "" || seen[t] || stopBigrams[t] || len(terms) >= MaxTerms {
			return
		}
		seen[t] = true
		terms = append(terms, t)
	}
	var ascii []rune
	var cjk []rune
	flushASCII := func() {
		if len(ascii) >= 2 {
			add(strings.ToLower(string(ascii)))
		}
		ascii = ascii[:0]
	}
	flushCJK := func() {
		switch {
		case len(cjk) < 2:
		case len(cjk) == 2:
			add(string(cjk))
		default:
			for i := 0; i+1 < len(cjk); i++ {
				add(string(cjk[i : i+2]))
			}
		}
		cjk = cjk[:0]
	}
	for _, r := range query {
		switch {
		case unicode.Is(unicode.Han, r):
			flushASCII()
			cjk = append(cjk, r)
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			flushCJK()
			ascii = append(ascii, r)
		default:
			flushASCII()
			flushCJK()
		}
	}
	flushASCII()
	flushCJK()
	return terms
}

// LikePatterns turns terms into escaped ILIKE patterns ("%term%").
func LikePatterns(terms []string) []string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	out := make([]string, 0, len(terms))
	for _, t := range terms {
		out = append(out, "%"+r.Replace(t)+"%")
	}
	return out
}

// Snippet returns up to width runes of content around the first term match.
func Snippet(content string, terms []string, width int) string {
	runes := []rune(content)
	if len(runes) <= width {
		return strings.TrimSpace(content)
	}
	lower := strings.ToLower(content)
	pos := -1
	for _, t := range terms {
		if i := strings.Index(lower, t); i >= 0 && (pos < 0 || i < pos) {
			pos = i
		}
	}
	start := 0
	if pos > 0 {
		start = len([]rune(lower[:pos])) - width/4
		if start < 0 {
			start = 0
		}
	}
	end := start + width
	if end > len(runes) {
		end = len(runes)
		start = end - width
	}
	out := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		out = "…" + out
	}
	if end < len(runes) {
		out += "…"
	}
	return out
}
