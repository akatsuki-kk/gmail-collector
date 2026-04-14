package gmailclient

import (
	"encoding/base64"
	"testing"

	"google.golang.org/api/gmail/v1"
)

// プレーンテキスト本文を優先して抽出することを確認する。
func TestExtractBodyPrefersPlainText(t *testing.T) {
	part := &gmail.MessagePart{
		MimeType: "multipart/alternative",
		Parts: []*gmail.MessagePart{
			{
				MimeType: "text/plain",
				Body:     &gmail.MessagePartBody{Data: encode("plain body")},
			},
			{
				MimeType: "text/html",
				Body:     &gmail.MessagePartBody{Data: encode("<p>html body</p>")},
			},
		},
	}

	got, err := extractBody(part)
	if err != nil {
		t.Fatalf("extractBody() returned error: %v", err)
	}
	if got != "plain body" {
		t.Fatalf("extractBody() = %q, want %q", got, "plain body")
	}
}

// HTML 本文をテキスト化して抽出対象にできることを確認する。
func TestExtractBodyConvertsHTMLToText(t *testing.T) {
	part := &gmail.MessagePart{
		MimeType: "text/html",
		Body:     &gmail.MessagePartBody{Data: encode("<div>お客様の<strong>ID</strong>は ABC123 です。</div>")},
	}

	got, err := extractBody(part)
	if err != nil {
		t.Fatalf("extractBody() returned error: %v", err)
	}
	if got != "お客様の ID は ABC123 です。" {
		t.Fatalf("extractBody() = %q, want %q", got, "お客様の ID は ABC123 です。")
	}
}

func encode(value string) string {
	return base64.URLEncoding.EncodeToString([]byte(value))
}
