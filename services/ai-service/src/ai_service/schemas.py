"""Request/response models for the internal API (see docs/ARCHITECTURE.md)."""

from __future__ import annotations

from enum import StrEnum
from typing import Annotated

from pydantic import BaseModel, ConfigDict, Field, StringConstraints

MAX_TEXT = 10_000
MAX_TITLE = 300
MAX_CRITERIA = 20

Title = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=MAX_TITLE)]
LongText = Annotated[str, StringConstraints(max_length=MAX_TEXT)]
RequiredLongText = Annotated[
    str, StringConstraints(strip_whitespace=True, min_length=1, max_length=MAX_TEXT)
]


class _Model(BaseModel):
    model_config = ConfigDict(extra="forbid")


class Policy(StrEnum):
    EXPLAIN_ONLY = "explain_only"
    FEEDBACK = "feedback"
    FULL = "full"


class TutorRequest(_Model):
    course_name: Title
    task_title: Title
    task_instructions: LongText = ""
    question: RequiredLongText
    policy: Policy


class TutorResponse(_Model):
    answer: str
    provider: str
    model: str


class Criterion(_Model):
    id: Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=100)]
    title: Title
    description: Annotated[str, StringConstraints(max_length=2_000)] = ""


class FeedbackRequest(_Model):
    task_title: Title
    submission_text: RequiredLongText
    criteria: Annotated[list[Criterion], Field(min_length=1, max_length=MAX_CRITERIA)]


class FeedbackItem(_Model):
    criterion_id: str
    comment: str


class FeedbackResponse(_Model):
    items: list[FeedbackItem]
    provider: str
    model: str
