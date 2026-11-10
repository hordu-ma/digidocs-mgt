from app.services.context_budget import (
    chunk,
    passages_from_hits,
    select_passages,
    split_for_summary,
    terms,
)


def test_terms_match_go_rules() -> None:
    # Same expectation as backend-go/internal/textsearch TestTermsChineseBigramsAndASCII.
    assert terms("请问 PPO 算法的奖励函数是什么？用了 GPU 吗") == [
        "ppo", "算法", "法的", "的奖", "奖励", "励函", "函数", "数是", "用了", "gpu",
    ]


def test_chunk_counts_match_go_and_migration() -> None:
    for n in (1, 800, 801, 1500, 1501, 5000):
        pieces = chunk("字" * n)
        assert len(pieces) == max(1, (n - 101) // 700 + 1)
        assert (len(pieces) - 1) * 700 + len(pieces[-1]) == n


def test_short_text_is_passed_whole() -> None:
    out = select_passages("短文本", "任何问题", 1000, title="t")
    assert len(out) == 1 and out[0].content == "短文本" and out[0].label == "片段1"


def test_long_text_keeps_relevant_chunks_within_budget() -> None:
    text = "背景。" * 1000 + "奖励函数由距离惩罚构成。" + "细节。" * 1000
    out = select_passages(text, "奖励函数怎么设计", 1600)
    assert sum(len(p.content) for p in out) <= 1600
    assert any("奖励函数" in p.content for p in out)
    assert [p.label for p in out] == [f"片段{i + 1}" for i in range(len(out))]
    assert [p.chunk_no for p in out] == sorted(p.chunk_no for p in out)


def test_no_match_falls_back_to_opening() -> None:
    text = "开头部分。" * 400 + "结尾。" * 400
    out = select_passages(text, "总结", 800)
    assert out[0].chunk_no == 0


def test_hits_respect_budget_and_skip_empty() -> None:
    hits = [
        {"content": "a" * 500, "title": "A", "document_id": "d1", "version_no": 2, "chunk_no": 3},
        {"content": ""},
        {"content": "b" * 600, "title": "B"},
        {"content": "c" * 300, "title": "C"},
    ]
    out = passages_from_hits(hits, 900)
    assert [p.title for p in out] == ["A", "C"]
    assert out[0].as_citation()["version_no"] == 2


def test_split_for_summary_samples_evenly() -> None:
    pieces, sampled = split_for_summary("x" * 1000, 100, 4)
    assert sampled and len(pieces) == 4
    pieces, sampled = split_for_summary("x" * 300, 100, 4)
    assert not sampled and len(pieces) == 3
