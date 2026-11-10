package textsearch

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestChunkMatchesMigrationBackfill(t *testing.T) {
	for _, n := range []int{1, 800, 801, 1500, 1501, 5000} {
		text := strings.Repeat("字", n)
		chunks := Chunk(text)
		// Same count as migration 014: GREATEST((L-101)/700, 0) + 1.
		want := (n-101)/700 + 1
		if n <= 101 {
			want = 1
		}
		if len(chunks) != want {
			t.Fatalf("len=%d chunks=%d want %d", n, len(chunks), want)
		}
		last := chunks[len(chunks)-1]
		if got := utf8.RuneCountInString(chunks[0]); got > ChunkSize {
			t.Fatalf("chunk too long: %d", got)
		}
		if (len(chunks)-1)*ChunkStep+utf8.RuneCountInString(last) != n {
			t.Fatalf("len=%d: chunks do not cover the text exactly", n)
		}
	}
	if Chunk("   ") != nil {
		t.Fatal("blank text yields no chunks")
	}
}

func TestTermsChineseBigramsAndASCII(t *testing.T) {
	got := Terms("请问 PPO 算法的奖励函数是什么？用了 GPU 吗")
	want := []string{"ppo", "算法", "法的", "的奖", "奖励", "励函", "函数", "数是", "用了", "gpu"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	if len(Terms(strings.Repeat("甲乙丙丁戊己庚辛壬癸", 10))) > MaxTerms {
		t.Fatal("terms must be capped")
	}
}

func TestLikePatternsEscape(t *testing.T) {
	got := LikePatterns([]string{"50%", "a_b"})
	if got[0] != `%50\%%` || got[1] != `%a\_b%` {
		t.Fatalf("got %v", got)
	}
}

func TestSnippetCentersOnMatch(t *testing.T) {
	content := strings.Repeat("前", 300) + "奖励函数" + strings.Repeat("后", 300)
	s := Snippet(content, []string{"奖励"}, 80)
	if !strings.Contains(s, "奖励函数") || !strings.HasPrefix(s, "…") || !strings.HasSuffix(s, "…") {
		t.Fatalf("snippet %q", s)
	}
}
