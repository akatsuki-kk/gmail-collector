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

// 本文に必要な文字列がすべて含まれる場合のみ一致とみなすことを確認する。
func TestContainsAll(t *testing.T) {
	body := "Your offer has been booked. Booking reference is GYG123."

	if !containsAll(body, []string{"Your offer has been booked", "Booking reference"}) {
		t.Fatal("containsAll() = false, want true")
	}

	if containsAll(body, []string{"Your offer has been booked", "A booking has been canceled"}) {
		t.Fatal("containsAll() = true, want false")
	}
}

func encode(value string) string {
	return base64.URLEncoding.EncodeToString([]byte(value))
}
