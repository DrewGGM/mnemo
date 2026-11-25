"""The LongMemEval judge: the official prompt of each question type, asked to a chat model.

Provenance: the templates are copied verbatim from get_anscheck_prompt in
src/evaluation/evaluate_qa.py of https://github.com/xiaowu0162/LongMemEval
(MIT license), at commit 9e0b455f4ef0e2ab8f2e582289761153549043fc of
2026-05-11; the file last changed in commit d6dc8b5 of 2024-11-24, and its
SHA-256 at that commit is
ecce9c4c79dc89d99534ac17b383a5cbb5b9f0c69ee98adaf0684742e3d95251.
As in that script, the judge gets the prompt as one user message with
temperature 0 and at most 10 tokens, and a reply containing "yes" in lower
case is a correct answer.
"""

from mnemo_eval.datasets.longmemeval import Question
from mnemo_eval.models import ChatModel

MAX_TOKENS = 10

_ANSWER_CHECK = (
    "I will give you a question, a correct answer, and a response from a model. Please answer "
    "yes if the response contains the correct answer. Otherwise, answer no. If the response is "
    "equivalent to the correct answer or contains all the intermediate steps to get the correct "
    "answer, you should also answer yes. If the response only contains a subset of the "
    "information required by the answer, answer no. "
)
_QUESTION_ANSWER_RESPONSE = (
    "\n\nQuestion: {}\n\nCorrect Answer: {}\n\nModel Response: {}\n\n"
    "Is the model response correct? Answer yes or no only."
)

TEMPLATES = {
    "single-session-user": _ANSWER_CHECK + _QUESTION_ANSWER_RESPONSE,
    "single-session-assistant": _ANSWER_CHECK + _QUESTION_ANSWER_RESPONSE,
    "multi-session": _ANSWER_CHECK + _QUESTION_ANSWER_RESPONSE,
    "temporal-reasoning": _ANSWER_CHECK
    + "In addition, do not penalize off-by-one errors for the number of days. If the question "
    "asks for the number of days/weeks/months, etc., and the model makes off-by-one errors "
    "(e.g., predicting 19 days when the answer is 18), the model's response is still correct. "
    + _QUESTION_ANSWER_RESPONSE,
    "knowledge-update": "I will give you a question, a correct answer, and a response from a "
    "model. Please answer yes if the response contains the correct answer. Otherwise, answer "
    "no. If the response contains some previous information along with an updated answer, the "
    "response should be considered as correct as long as the updated answer is the required "
    "answer." + _QUESTION_ANSWER_RESPONSE,
    "single-session-preference": "I will give you a question, a rubric for desired personalized "
    "response, and a response from a model. Please answer yes if the response satisfies the "
    "desired response. Otherwise, answer no. The model does not need to reflect all the points "
    "in the rubric. The response is correct as long as it recalls and utilizes the user's "
    "personal information correctly.\n\nQuestion: {}\n\nRubric: {}\n\nModel Response: {}\n\n"
    "Is the model response correct? Answer yes or no only.",
}
ABSTENTION_TEMPLATE = (
    "I will give you an unanswerable question, an explanation, and a response from a model. "
    "Please answer yes if the model correctly identifies the question as unanswerable. The model "
    "could say that the information is incomplete, or some other information is given but the "
    "asked information is not.\n\nQuestion: {}\n\nExplanation: {}\n\nModel Response: {}\n\n"
    "Does the model correctly identify the question as unanswerable? Answer yes or no only."
)


def prompt(question: Question, answer: str) -> str:
    """The official prompt for the question's type, or the abstention prompt."""
    if question.is_abstention:
        template = ABSTENTION_TEMPLATE
    elif question.type in TEMPLATES:
        template = TEMPLATES[question.type]
    else:
        raise ValueError(f"question {question.id}: no judge prompt for type {question.type!r}")
    return template.format(question.question, question.answer, answer)


def judge(question: Question, answer: str, model: ChatModel) -> bool:
    """Asks model whether answer is correct for question."""
    reply = model.complete(prompt(question, answer), max_tokens=MAX_TOKENS)
    return "yes" in reply.text.lower()
