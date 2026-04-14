package app

import (
	"testing"

	"github.com/akatsuki-kk/gmail-collector/internal/gmailclient"
)

// 進捗表示の文言を期待どおりに整形することを確認する。
func TestFormatProgress(t *testing.T) {
	got := formatProgress(gmailclient.Progress{
		Total:     12,
		Processed: 5,
		Matched:   3,
	})

	want := "処理中: 5/12件 (抽出対象: 3件)"
	if got != want {
		t.Fatalf("formatProgress() = %q, want %q", got, want)
	}
}
