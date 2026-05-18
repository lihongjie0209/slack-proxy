package dingtalk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// ---- buildMarkdown ----

func TestBuildMarkdown_TextOnly(t *testing.T) {
	msg := &SlackMessage{Text: "hello world"}
	got := buildMarkdown(msg)
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestBuildMarkdown_WithAttachment(t *testing.T) {
	msg := &SlackMessage{
		Text: "alert",
		Attachments: []Attachment{
			{Title: "My Title", Text: "details here"},
		},
	}
	got := buildMarkdown(msg)
	if !strings.Contains(got, "#### My Title") {
		t.Errorf("missing title heading in: %q", got)
	}
	if !strings.Contains(got, "details here") {
		t.Errorf("missing attachment text in: %q", got)
	}
}

func TestBuildMarkdown_FallbackWhenNoText(t *testing.T) {
	msg := &SlackMessage{
		Attachments: []Attachment{
			{Fallback: "fallback text"},
		},
	}
	got := buildMarkdown(msg)
	if !strings.Contains(got, "fallback text") {
		t.Errorf("expected fallback text in: %q", got)
	}
}

func TestBuildMarkdown_WithFields(t *testing.T) {
	msg := &SlackMessage{
		Attachments: []Attachment{
			{
				Title: "Alert",
				Fields: []Field{
					{Title: "env", Value: "prod"},
					{Title: "severity", Value: "high"},
				},
			},
		},
	}
	got := buildMarkdown(msg)
	if !strings.Contains(got, "**env**: prod") {
		t.Errorf("missing field in: %q", got)
	}
	if !strings.Contains(got, "**severity**: high") {
		t.Errorf("missing field in: %q", got)
	}
}

func TestBuildMarkdown_Empty(t *testing.T) {
	got := buildMarkdown(&SlackMessage{})
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ---- firstLine ----

func TestFirstLine_SingleLine(t *testing.T) {
	got := firstLine("hello")
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestFirstLine_MultiLine(t *testing.T) {
	got := firstLine("line1\nline2\nline3")
	if got != "line1" {
		t.Errorf("got %q, want %q", got, "line1")
	}
}

func TestFirstLine_Empty(t *testing.T) {
	got := firstLine("")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFirstLine_LongLine(t *testing.T) {
	long := strings.Repeat("a", 100)
	got := firstLine(long)
	if len(got) != 80 {
		t.Errorf("expected truncation to 80, got len=%d", len(got))
	}
}

func TestFirstLine_TrimSpace(t *testing.T) {
	got := firstLine("  spaced  \nmore")
	if got != "spaced" {
		t.Errorf("got %q, want %q", got, "spaced")
	}
}

// ---- signURLWithTimestamp ----

func TestSignURLWithTimestamp_ContainsParams(t *testing.T) {
	result, err := signURLWithTimestamp("https://example.com/robot/send?access_token=TOKEN", "mysecret", "1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "timestamp=1234567890") {
		t.Errorf("missing timestamp in: %s", result)
	}
	if !strings.Contains(result, "sign=") {
		t.Errorf("missing sign in: %s", result)
	}
}

func TestSignURLWithTimestamp_SignatureCorrect(t *testing.T) {
	secret := "testsecret"
	timestamp := "1700000000000"

	result, err := signURLWithTimestamp("https://example.com?access_token=T", secret, timestamp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Recompute expected signature
	strToSign := timestamp + "\n" + secret
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strToSign))
	expectedSign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	if !strings.Contains(result, "sign="+expectedSign) {
		t.Errorf("signature mismatch in URL: %s", result)
	}
}

func TestSignURLWithTimestamp_NoQuerySep(t *testing.T) {
	result, err := signURLWithTimestamp("https://example.com/send", "s", "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should use ? not &
	if !strings.Contains(result, "?timestamp=") {
		t.Errorf("expected ? separator in: %s", result)
	}
}

// ---- Send (integration with mock HTTP server) ----

func TestSend_Success(t *testing.T) {
	var received dingTalkRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	msg := &SlackMessage{
		Text: "deploy succeeded",
		Attachments: []Attachment{
			{Title: "Details", Text: "v1.2.3 deployed to prod"},
		},
	}
	if err := Send(srv.URL, "", msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.MsgType != "markdown" {
		t.Errorf("msgtype = %q, want markdown", received.MsgType)
	}
	if received.Markdown.Title != "deploy succeeded" {
		t.Errorf("title = %q, want %q", received.Markdown.Title, "deploy succeeded")
	}
	if !strings.Contains(received.Markdown.Text, "v1.2.3") {
		t.Errorf("body missing attachment text: %q", received.Markdown.Text)
	}
}

func TestSend_TitleFallsBackToAttachment(t *testing.T) {
	var received dingTalkRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	msg := &SlackMessage{
		Attachments: []Attachment{{Title: "Attachment Title", Text: "body"}},
	}
	if err := Send(srv.URL, "", msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Markdown.Title != "Attachment Title" {
		t.Errorf("title = %q, want %q", received.Markdown.Title, "Attachment Title")
	}
}

func TestSend_DefaultTitle(t *testing.T) {
	var received dingTalkRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	if err := Send(srv.URL, "", &SlackMessage{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Markdown.Title != "Notification" {
		t.Errorf("title = %q, want Notification", received.Markdown.Title)
	}
}

func TestSend_WithSecret_HasSignParams(t *testing.T) {
	var reqURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqURL = r.URL.RawQuery
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	if err := Send(srv.URL, "mysecret", &SlackMessage{Text: "hi"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(reqURL, "timestamp=") || !strings.Contains(reqURL, "sign=") {
		t.Errorf("expected sign params in query, got: %s", reqURL)
	}
}

func TestSend_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := Send(srv.URL, "", &SlackMessage{Text: "hi"})
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestSend_DingTalkErrCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"errcode":310000,"errmsg":"invalid token"}`))
	}))
	defer srv.Close()

	err := Send(srv.URL, "", &SlackMessage{Text: "hi"})
	if err == nil {
		t.Fatal("expected error for errcode!=0, got nil")
	}
	if !strings.Contains(err.Error(), "310000") {
		t.Errorf("error should contain errcode, got: %v", err)
	}
}

func TestSend_ConnectionRefused(t *testing.T) {
	err := Send("http://127.0.0.1:1", "", &SlackMessage{Text: "hi"})
	if err == nil {
		t.Fatal("expected error for refused connection, got nil")
	}
}
