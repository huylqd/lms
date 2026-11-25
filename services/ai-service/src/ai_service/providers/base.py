"""Provider interface every AI backend must implement."""

from __future__ import annotations

from abc import ABC, abstractmethod

from ..schemas import FeedbackItem, FeedbackRequest, TutorRequest


class ProviderError(RuntimeError):
    """Configuration or selection error for an AI provider."""


class AIProvider(ABC):
    """Abstract AI backend.

    Implementations must be safe to call concurrently and must not log
    request content (it contains student work).
    """

    @property
    @abstractmethod
    def name(self) -> str:
        """Short provider id returned to callers, e.g. ``"mock"``."""

    @property
    @abstractmethod
    def model(self) -> str:
        """Model identifier returned to callers, e.g. ``"mock-1"``."""

    @abstractmethod
    async def tutor(self, request: TutorRequest) -> str:
        """Return the tutor answer text, respecting ``request.policy``."""

    @abstractmethod
    async def feedback(self, request: FeedbackRequest) -> list[FeedbackItem]:
        """Return exactly one item per criterion, in the same order."""
