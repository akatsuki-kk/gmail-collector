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

// 複数検索結果を message_id 単位で重複なく結合することを確認する。
func TestMergeResults(t *testing.T) {
	current := []gmailclient.Result{
		{MessageID: "1", Subject: "Booking"},
	}
	incoming := []gmailclient.Result{
		{MessageID: "1", Subject: "Booking"},
		{MessageID: "2", Subject: "New booking received"},
	}

	got := mergeResults(current, incoming)
	if len(got) != 2 {
		t.Fatalf("mergeResults() len = %d, want %d", len(got), 2)
	}
	if got[1].MessageID != "2" {
		t.Fatalf("mergeResults()[1].MessageID = %q, want %q", got[1].MessageID, "2")
	}
}
