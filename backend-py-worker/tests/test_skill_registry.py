from app.services.skill_registry import SkillRegistry, SkillResolutionError


def test_registry_defaults_to_first_allowlisted_skill() -> None:
    registry = SkillRegistry()

    definition = registry.resolve("assistant.ask", "")

    assert definition.name == "answer_with_context"
    assert definition.version == "v1"


def test_registry_rejects_task_without_allowlist() -> None:
    registry = SkillRegistry()

    try:
        registry.resolve("document.extract_text", "")
    except SkillResolutionError as exc:
        assert "未配置可用 skill" in str(exc)
    else:
        raise AssertionError("expected SkillResolutionError")


def test_registry_rejects_unregistered_allowlisted_skill() -> None:
    registry = SkillRegistry()
    registry._allowlists["assistant.ask"] = ("missing_skill",)

    try:
        registry.resolve("assistant.ask", "")
    except SkillResolutionError as exc:
        assert "未注册" in str(exc)
    else:
        raise AssertionError("expected SkillResolutionError")


def test_registry_rejects_skill_not_supporting_task() -> None:
    registry = SkillRegistry()
    registry._allowlists["assistant.ask"] = ("document_summary",)

    try:
        registry.resolve("assistant.ask", "document_summary")
    except SkillResolutionError as exc:
        assert "不支持任务" in str(exc)
    else:
        raise AssertionError("expected SkillResolutionError")
