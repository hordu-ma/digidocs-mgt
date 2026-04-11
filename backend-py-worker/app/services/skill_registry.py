from __future__ import annotations

from dataclasses import dataclass
from typing import cast

from ..core.config import settings
from ..tasks.contracts import TaskType


class SkillResolutionError(RuntimeError):
    """Raised when a requested skill is not allowed for the current task."""


@dataclass(frozen=True, slots=True)
class SkillDefinition:
    name: str
    version: str
    task_types: tuple[TaskType, ...]
    description: str


class SkillRegistry:
    def __init__(self) -> None:
        self._definitions: dict[str, SkillDefinition] = {
            "answer_with_context": SkillDefinition(
                name="answer_with_context",
                version="v1",
                task_types=("assistant.ask",),
                description="基于显式 scope/context/memory 进行问答。",
            ),
            "document_summary": SkillDefinition(
                name="document_summary",
                version="v1",
                task_types=("document.summarize",),
                description="基于显式上下文生成文档摘要和建议。",
            ),
            "handover_summary": SkillDefinition(
                name="handover_summary",
                version="v1",
                task_types=("handover.summarize",),
                description="基于显式上下文生成交接摘要和待确认事项。",
            ),
            "structured_suggestion": SkillDefinition(
                name="structured_suggestion",
                version="v1",
                task_types=("assistant.generate_suggestion",),
                description="基于显式上下文生成结构化建议。",
            ),
        }
        self._allowlists: dict[TaskType, tuple[str, ...]] = {
            "assistant.ask": _parse_csv(settings.openclaw_skills_assistant_ask),
            "document.summarize": _parse_csv(settings.openclaw_skills_document_summarize),
            "handover.summarize": _parse_csv(settings.openclaw_skills_handover_summarize),
            "assistant.generate_suggestion": _parse_csv(
                settings.openclaw_skills_generate_suggestion
            ),
            "document.extract_text": tuple(),
        }

    def resolve(self, task_type: TaskType, requested_name: str) -> SkillDefinition:
        allowed = self._allowlists.get(task_type, tuple())
        if len(allowed) == 0:
            raise SkillResolutionError(f"任务 {task_type} 未配置可用 skill")

        skill_name = requested_name.strip() if requested_name.strip() != "" else allowed[0]
        if skill_name not in allowed:
            raise SkillResolutionError(f"skill {skill_name} 不在任务 {task_type} 的白名单内")

        definition = self._definitions.get(skill_name)
        if definition is None:
            raise SkillResolutionError(f"skill {skill_name} 未注册")
        if task_type not in definition.task_types:
            raise SkillResolutionError(f"skill {skill_name} 不支持任务 {task_type}")
        return definition


def _parse_csv(raw: str) -> tuple[str, ...]:
    items: list[str] = []
    for part in raw.split(","):
        item = part.strip()
        if item != "":
            items.append(item)
    return cast(tuple[str, ...], tuple(items))
