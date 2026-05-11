package output

import "testing"

func TestFormatRows_TruncatesLongTextAndDetectsHasMore(t *testing.T) {
	rows := [][]any{
		{1, "short"},
		{2, "this is a very long value that should be truncated for agent output"},
		{3, "overflow row"},
	}

	res := FormatRows([]string{"id", "body"}, rows, 1, 2, 16)

	if !res.Truncated {
		t.Fatal("expected truncated output")
	}
	if !res.HasMore {
		t.Fatal("expected has_more true")
	}
	if res.NextPage == nil || *res.NextPage != 2 {
		t.Fatalf("expected next_page 2, got %#v", res.NextPage)
	}
	if got := len(res.Rows); got != 2 {
		t.Fatalf("expected 2 rows, got %d", got)
	}
	if got := res.Rows[1][1]; got != "this is a very l..." {
		t.Fatalf("expected truncated cell, got %#v", got)
	}
}

func TestFormatRows_ReplacesBinaryValues(t *testing.T) {
	res := FormatRows([]string{"blob"}, [][]any{{[]byte{1, 2, 3, 4}}}, 1, 10, 16)

	if !res.Truncated {
		t.Fatal("expected binary value to mark result truncated")
	}
	if res.TruncatedCells != 1 {
		t.Fatalf("expected 1 truncated cell, got %d", res.TruncatedCells)
	}
	if got := res.Rows[0][0]; got != "[BINARY 4B]" {
		t.Fatalf("expected binary placeholder, got %#v", got)
	}
}
