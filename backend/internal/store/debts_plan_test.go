package store

import "testing"

func TestSplitAmountsRemainder(t *testing.T) {
	got, err := splitAmounts(30_000_000, 11_500_000, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{11_500_000, 11_500_000, 7_000_000}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSplitAmountsEven(t *testing.T) {
	got, err := splitAmounts(30_000_000, 10_000_000, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != 10_000_000 || got[2] != 10_000_000 {
		t.Fatalf("got %v", got)
	}
}
