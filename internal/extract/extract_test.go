package extract

import "testing"

// キャプチャグループの先頭要素を抽出値として返すことを確認する。
func TestApplyUsesFirstCaptureGroup(t *testing.T) {
	body := "お客様のIDは（ABC123）、電話番号は 000-0000-9999 です。"
	rules := map[string]string{
		"id":           `IDは（(.+?)）`,
		"phone_number": `電話番号は ([0-9-]+)`,
	}

	got, err := Apply(body, rules)
	if err != nil {
		t.Fatalf("Apply() returned error: %v", err)
	}

	if got["id"] != "ABC123" {
		t.Fatalf("id = %q, want %q", got["id"], "ABC123")
	}
	if got["phone_number"] != "000-0000-9999" {
		t.Fatalf("phone_number = %q, want %q", got["phone_number"], "000-0000-9999")
	}
}

// 複数回一致しても最初の一致だけを採用することを確認する。
func TestApplyReturnsFirstMatchOnly(t *testing.T) {
	body := "注文番号 A-001 / 注文番号 A-002"
	rules := map[string]string{
		"order_id": `注文番号 ([A-Z]-[0-9]+)`,
	}

	got, err := Apply(body, rules)
	if err != nil {
		t.Fatalf("Apply() returned error: %v", err)
	}

	if got["order_id"] != "A-001" {
		t.Fatalf("order_id = %q, want %q", got["order_id"], "A-001")
	}
}
