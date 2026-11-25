import pytest

from mnemo_eval import metrics


def row(qtype="multi-session", correct=True, abstention=False, evidence=(), retrieved=()):
    return {
        "type": qtype,
        "correct": correct,
        "is_abstention": abstention,
        "evidence_session_ids": list(evidence),
        "retrieved_session_ids": list(retrieved),
    }


def test_accuracy_counts_correct_rows():
    rows = [row(correct=True), row(correct=False), row(correct=True), row(correct=True)]
    assert metrics.accuracy(rows) == pytest.approx(0.75)


def test_accuracy_by_type_splits_rows_by_type():
    rows = [
        row("temporal-reasoning", True),
        row("temporal-reasoning", False),
        row("knowledge-update", True),
        row("multi-session", False),
    ]
    assert metrics.accuracy_by_type(rows) == {
        "knowledge-update": 1.0,
        "multi-session": 0.0,
        "temporal-reasoning": 0.5,
    }


def test_accuracy_of_no_rows_is_an_error():
    with pytest.raises(ValueError, match="no rows"):
        metrics.accuracy([])


def test_recall_at_k_needs_every_evidence_session_in_top_k():
    rows = [
        row(evidence=["a"], retrieved=["x", "a"]),
        row(evidence=["a", "b"], retrieved=["a", "x", "b"]),
    ]
    assert [metrics.recall_at_k(rows, k) for k in (2, 3)] == [0.5, 1.0]


def test_recall_at_k_counts_distinct_sessions():
    # Episodes of one session come back as several entries; k counts sessions.
    rows = [row(evidence=["b"], retrieved=["a", "a", "a", "b"])]
    assert metrics.recall_at_k(rows, 2) == 1.0


def test_recall_at_k_excludes_abstention_questions():
    rows = [
        row(evidence=["a"], retrieved=["a"]),
        row(abstention=True, evidence=["a"], retrieved=["x"]),
    ]
    assert metrics.recall_at_k(rows, 5) == 1.0


def test_recall_at_k_without_answerable_rows_is_none():
    assert metrics.recall_at_k([row(abstention=True, evidence=["a"])], 5) is None


def test_mrr_uses_rank_of_first_evidence_session():
    rows = [
        row(evidence=["b"], retrieved=["a", "a", "b"]),
        row(evidence=["c"], retrieved=["a", "b"]),
        row(abstention=True, evidence=["a"], retrieved=["b", "a"]),
    ]
    assert metrics.mrr(rows) == pytest.approx(0.25)


def test_percentile_is_over_the_whole_list():
    values = [5.0, 1.0, 3.0, 2.0, 4.0]
    got = [metrics.percentile(values, p) for p in (0, 50, 95, 100)]
    assert got == pytest.approx([1.0, 3.0, 4.8, 5.0])


def test_percentile_of_two_lists_is_not_the_mean_of_their_percentiles():
    fast, slow = [1.0] * 9 + [100.0], [10.0] * 10
    whole = metrics.percentile(fast + slow, 95)
    mean_of_parts = (metrics.percentile(fast, 95) + metrics.percentile(slow, 95)) / 2
    assert whole == pytest.approx(14.5)
    assert mean_of_parts == pytest.approx(32.725)


def test_percentile_outside_range_is_rejected():
    with pytest.raises(ValueError, match="outside"):
        metrics.percentile([1.0], 101)


def test_bootstrap_ci_is_reproducible_with_its_seed():
    values = [float(i * i % 17) for i in range(25)]
    first = metrics.bootstrap_ci(values, seed=7, resamples=2000)
    assert first == metrics.bootstrap_ci(values, seed=7, resamples=2000)
    assert first != metrics.bootstrap_ci(values, seed=8, resamples=2000)


def test_bootstrap_ci_brackets_the_mean():
    values = [1.0] * 30 + [0.0] * 30
    low, high = metrics.bootstrap_ci(values, seed=1, resamples=2000)
    # The standard error of the mean is 0.065, so the interval is about 0.5 +- 0.13.
    assert [low, high] == pytest.approx([0.37, 0.63], abs=0.03)


def test_bootstrap_ci_of_constant_values_is_a_point():
    assert metrics.bootstrap_ci([1.0] * 10, seed=3, resamples=100) == (1.0, 1.0)
