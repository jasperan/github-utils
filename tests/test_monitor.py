from github_utils.analytics.monitor import ProgressState

def test_progress_state_init():
    state = ProgressState(total_commits=1000)
    assert state.completed == 0
    assert state.total_commits == 1000
    assert state.throughput == 0.0

def test_progress_state_update():
    state = ProgressState(total_commits=100)
    state.record_commit()
    state.record_commit()
    assert state.completed == 2

def test_progress_percentage():
    state = ProgressState(total_commits=200)
    for _ in range(50):
        state.record_commit()
    assert state.percentage == 25.0

def test_progress_state_push_tracking():
    state = ProgressState(total_commits=100)
    state.record_push()
    state.record_push()
    assert state.pushes == 2
