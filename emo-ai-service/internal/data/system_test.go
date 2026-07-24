package data

import "testing"

func TestTrimJSONQuoteDecodesJSONString(t *testing.T) {
	got := trimJSONQuote(`"第一行\n第二行 \"引用\""`)
	want := "第一行\n第二行 \"引用\""
	if got != want {
		t.Fatalf("trimJSONQuote() = %q, want %q", got, want)
	}
}

func TestTrimJSONQuoteKeepsNonStringJSON(t *testing.T) {
	const value = `{"title":"隐私"}`
	if got := trimJSONQuote(value); got != value {
		t.Fatalf("trimJSONQuote() = %q, want %q", got, value)
	}
}
