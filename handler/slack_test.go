package handler_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"slack-proxy/config"
	"slack-proxy/dingtalk"
	"slack-proxy/handler"
)

var testRoute = config.Route{
	SlackPath: "/hook/test",
	DingTalk:  config.DingTalkConfig{Webhook: "https://example.com", Secret: "s"},
}

func okSender(_ string, _ string, _ *dingtalk.SlackMessage) error { return nil }
func errSender(_ string, _ string, _ *dingtalk.SlackMessage) error {
	return errors.New("dingtalk unavailable")
}

func TestHandler_SuccessfulForward(t *testing.T) {
	var capturedMsg *dingtalk.SlackMessage
	sender := func(_ string, _ string, msg *dingtalk.SlackMessage) error {
		capturedMsg = msg
		return nil
	}

	body := `{"text":"hello","username":"bot"}`
	req := httptest.NewRequest(http.MethodPost, "/hook/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.NewSlackHandler(testRoute, sender)(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"ok":true`) {
		t.Errorf("body = %q, want ok:true", rr.Body.String())
	}
	if capturedMsg == nil || capturedMsg.Text != "hello" {
		t.Errorf("capturedMsg.Text = %q, want hello", capturedMsg.Text)
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/hook/test", nil)
		rr := httptest.NewRecorder()
		handler.NewSlackHandler(testRoute, okSender)(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("[%s] status = %d, want 405", method, rr.Code)
		}
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook/test", strings.NewReader(`{not json`))
	rr := httptest.NewRecorder()
	handler.NewSlackHandler(testRoute, okSender)(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestHandler_SenderError(t *testing.T) {
	body := `{"text":"alert"}`
	req := httptest.NewRequest(http.MethodPost, "/hook/test", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.NewSlackHandler(testRoute, errSender)(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rr.Code)
	}
}

func TestHandler_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook/test", bytes.NewReader([]byte{}))
	rr := httptest.NewRecorder()
	handler.NewSlackHandler(testRoute, okSender)(rr, req)
	// empty body is invalid JSON
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestHandler_NilSenderUsesDefault(t *testing.T) {
	// nil sender should not panic (uses dingtalk.Send default)
	h := handler.NewSlackHandler(testRoute, nil)
	if h == nil {
		t.Fatal("handler should not be nil")
	}
}

func TestHandler_WithAttachments(t *testing.T) {
	var capturedMsg *dingtalk.SlackMessage
	sender := func(_ string, _ string, msg *dingtalk.SlackMessage) error {
		capturedMsg = msg
		return nil
	}

	body := `{"text":"title","attachments":[{"title":"att","text":"body","fields":[{"title":"env","value":"prod"}]}]}`
	req := httptest.NewRequest(http.MethodPost, "/hook/test", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.NewSlackHandler(testRoute, sender)(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if len(capturedMsg.Attachments) != 1 {
		t.Errorf("attachments count = %d, want 1", len(capturedMsg.Attachments))
	}
	if capturedMsg.Attachments[0].Fields[0].Value != "prod" {
		t.Errorf("field value = %q, want prod", capturedMsg.Attachments[0].Fields[0].Value)
	}
}

func TestHandler_ContentTypeJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook/test", strings.NewReader(`{"text":"hi"}`))
	rr := httptest.NewRecorder()
	handler.NewSlackHandler(testRoute, okSender)(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}
