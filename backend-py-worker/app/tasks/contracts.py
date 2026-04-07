from dataclasses import dataclass, field
from typing import Literal

type ObjectDict = dict[str, object]


TaskType = Literal[
    "assistant.ask",
    "document.summarize",
    "handover.summarize",
    "document.extract_text",
    "assistant.generate_suggestion",
]


@dataclass(slots=True)
class WorkerTask:
    request_id: str
    task_type: TaskType
    related_type: str | None = None
    related_id: str | None = None
    payload: ObjectDict = field(default_factory=dict)


@dataclass(slots=True)
class TaskResult:
    request_id: str
    status: Literal["completed", "failed"]
    output: ObjectDict = field(default_factory=dict)
    error_message: str | None = None
