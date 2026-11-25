"""Deterministic offline provider used for development, tests and demos."""

from __future__ import annotations

from ..schemas import FeedbackItem, FeedbackRequest, Policy, TutorRequest
from .base import AIProvider

SELF_WORK_REMINDER = (
    "Lưu ý: bạn cần tự làm bài và tự kiểm chứng kết quả; "
    "trợ lý chỉ giải thích khái niệm, không làm hộ bài."
)


def _excerpt(text: str, limit: int = 120) -> str:
    text = " ".join(text.split())
    return text if len(text) <= limit else text[: limit - 1].rstrip() + "…"


class MockProvider(AIProvider):
    @property
    def name(self) -> str:
        return "mock"

    @property
    def model(self) -> str:
        return "mock-1"

    async def tutor(self, request: TutorRequest) -> str:
        context = f"Môn «{request.course_name}», bài «{request.task_title}»."
        question = f"Câu hỏi của bạn: “{_excerpt(request.question)}”."
        match request.policy:
            case Policy.EXPLAIN_ONLY:
                body = [
                    "Hãy bắt đầu bằng việc xác định khái niệm cốt lõi mà đề bài yêu cầu "
                    "và nhắc lại định nghĩa của nó bằng lời của chính bạn.",
                    "Một vài câu hỏi gợi mở:",
                    "1. Đề bài cho những dữ kiện gì và yêu cầu đầu ra là gì?",
                    "2. Khái niệm nào trong môn học liên quan trực tiếp đến yêu cầu này?",
                    "3. Bạn có thể thử với một ví dụ nhỏ, đơn giản trước không?",
                    "4. Làm sao để bạn biết kết quả của mình là đúng?",
                    SELF_WORK_REMINDER,
                ]
            case Policy.FEEDBACK:
                body = [
                    "Gợi ý để cải thiện bài làm:",
                    "- Đối chiếu lại từng yêu cầu của đề bài với phần bạn đã làm.",
                    "- Trình bày rõ lập luận ở mỗi bước, tránh nhảy cóc.",
                    "- Bổ sung ví dụ hoặc trường hợp kiểm thử để minh chứng.",
                    "- Rà soát lỗi chính tả, định dạng và trích dẫn nguồn.",
                    "Bạn vẫn là người chịu trách nhiệm cuối cùng cho bài làm của mình.",
                ]
            case Policy.FULL:
                body = [
                    "Hướng dẫn chi tiết từng bước:",
                    "Bước 1: Phân tích đề bài - liệt kê dữ kiện, ràng buộc và kết quả cần đạt.",
                    "Bước 2: Ôn lại kiến thức nền của môn học liên quan đến yêu cầu.",
                    "Bước 3: Lập dàn ý hoặc thuật toán giải, chia thành các phần nhỏ.",
                    "Bước 4: Thực hiện từng phần, kiểm tra kết quả trung gian.",
                    "Bước 5: Tổng hợp, kiểm chứng lại với yêu cầu ban đầu và rút ra nhận xét.",
                    "Hãy hiểu rõ từng bước trước khi áp dụng vào bài của bạn.",
                ]
        return "\n".join([context, question, *body])

    async def feedback(self, request: FeedbackRequest) -> list[FeedbackItem]:
        words = len(request.submission_text.split())
        items: list[FeedbackItem] = []
        for criterion in request.criteria:
            comment = (
                f"Tiêu chí «{criterion.title}» (bài «{request.task_title}»): "
                f"bài nộp có khoảng {words} từ. Hãy đối chiếu bài làm với mô tả tiêu chí"
                + (f" ({_excerpt(criterion.description, 80)})" if criterion.description else "")
                + ", bổ sung minh chứng cụ thể và giải thích rõ hơn những điểm còn sơ sài."
            )
            items.append(FeedbackItem(criterion_id=criterion.id, comment=comment))
        return items
