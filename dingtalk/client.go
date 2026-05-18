package dingtalk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SlackMessage represents the incoming Slack Incoming Webhook payload.
type SlackMessage struct {
	Text        string       `json:"text"`
	Username    string       `json:"username"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Title    string  `json:"title"`
	Text     string  `json:"text"`
	Color    string  `json:"color"`
	Fields   []Field `json:"fields"`
	Fallback string  `json:"fallback"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// dingTalkRequest is the payload sent to DingTalk robot.
type dingTalkRequest struct {
	MsgType  string          `json:"msgtype"`
	Markdown dingTalkMarkdown `json:"markdown"`
}

type dingTalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Send converts a Slack message to DingTalk markdown and sends it.
func Send(webhook, secret string, msg *SlackMessage) error {
	body := buildMarkdown(msg)
	title := firstLine(msg.Text)
	if title == "" && len(msg.Attachments) > 0 {
		title = msg.Attachments[0].Title
	}
	if title == "" {
		title = "Notification"
	}

	reqBody, err := json.Marshal(dingTalkRequest{
		MsgType:  "markdown",
		Markdown: dingTalkMarkdown{Title: title, Text: body},
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	targetURL := webhook
	if secret != "" {
		targetURL, err = signURL(webhook, secret)
		if err != nil {
			return fmt.Errorf("sign url: %w", err)
		}
	}

	resp, err := http.Post(targetURL, "application/json", bytes.NewReader(reqBody)) //nolint:noctx
	if err != nil {
		return fmt.Errorf("post to dingtalk: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk returned %d: %s", resp.StatusCode, respBytes)
	}

	// DingTalk returns errcode!=0 on failure even with 200 status.
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(respBytes, &result); err == nil && result.ErrCode != 0 {
		return fmt.Errorf("dingtalk error %d: %s", result.ErrCode, result.ErrMsg)
	}
	return nil
}

// buildMarkdown converts a Slack message to DingTalk markdown text.
func buildMarkdown(msg *SlackMessage) string {
	var sb strings.Builder

	if msg.Text != "" {
		sb.WriteString(msg.Text)
		sb.WriteString("\n\n")
	}

	for _, att := range msg.Attachments {
		if att.Title != "" {
			sb.WriteString("#### ")
			sb.WriteString(att.Title)
			sb.WriteString("\n")
		}
		text := att.Text
		if text == "" {
			text = att.Fallback
		}
		if text != "" {
			sb.WriteString(text)
			sb.WriteString("\n")
		}
		for _, f := range att.Fields {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", f.Title, f.Value))
		}
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

// signURL appends DingTalk HMAC-SHA256 signature params to the webhook URL.
func signURL(webhook, secret string) (string, error) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	return signURLWithTimestamp(webhook, secret, timestamp)
}

// signURLWithTimestamp is the testable core of signURL.
func signURLWithTimestamp(webhook, secret, timestamp string) (string, error) {
	strToSign := timestamp + "\n" + secret

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strToSign))
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	sep := "&"
	if !strings.Contains(webhook, "?") {
		sep = "?"
	}
	return fmt.Sprintf("%s%stimestamp=%s&sign=%s", webhook, sep, timestamp, sign), nil
}

// firstLine returns the first non-empty line of s.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	if len(s) > 80 {
		return s[:80]
	}
	return s
}
