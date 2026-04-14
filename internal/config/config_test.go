package config

import "testing"

// 正常な検索条件と抽出条件を受け付けることを確認する。
func TestConfigValidateAcceptsValidConfig(t *testing.T) {
	cfg := Config{
		Search: SearchConfig{
			From:            []string{"example@example.com"},
			SubjectContains: []string{"お客様のID"},
		},
		Extract: map[string]string{
			"id": `IDは（(.+?)）`,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
}

// 抽出条件が空の場合にエラーになることを確認する。
func TestConfigValidateRejectsEmptyExtract(t *testing.T) {
	cfg := Config{
		Search: SearchConfig{
			From: []string{"example@example.com"},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}

// 検索条件から Gmail クエリ文字列を組み立てることを確認する。
func TestSearchConfigBuildQuery(t *testing.T) {
	search := SearchConfig{
		From:            []string{"foo@example.com"},
		SubjectContains: []string{"お客様 ID"},
		After:           "2025/01/01",
		Before:          "2025/12/31",
		Label:           []string{"inbox"},
	}

	got := search.BuildQuery()
	want := `from:foo@example.com subject:"お客様 ID" after:2025/01/01 before:2025/12/31 label:inbox`

	if got != want {
		t.Fatalf("BuildQuery() = %q, want %q", got, want)
	}
}
