from app import app


def test_healthz():
    response = app.test_client().get("/healthz")
    assert response.status_code == 200
    assert response.text == "ok"
