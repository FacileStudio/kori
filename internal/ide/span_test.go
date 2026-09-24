package ide

import "testing"

func TestSpanOfRewriteMarksTheChangedLines(t *testing.T) {
	cases := []struct {
		name          string
		before, after string
		first, last   int
	}{
		{"create a file", "", "hello\n", 1, 1},
		{"append a line", "one\ntwo\n", "one\ntwo\nthree\n", 3, 3},
		{"replace one line", "one\ntwo\nthree\n", "one\nTWO\nthree\n", 2, 2},
		{"replace a block", "a\nb\nc\nd\n", "a\nX\nY\nd\n", 2, 3},
		{"remove two lines", "a\nb\nc\nd\n", "a\nd\n", 2, 2},
		{"rewrite everything", "a\nb\n", "x\ny\n", 1, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first, last := spanOfRewrite(tc.before, tc.after)
			if first != tc.first || last != tc.last {
				t.Errorf("spanOfRewrite = %d..%d, want %d..%d", first, last, tc.first, tc.last)
			}
		})
	}
}

func TestSpanOfFragmentLocatesItInTheFile(t *testing.T) {
	text := "one\ntwo\nthree\nfour\n"
	first, last, ok := spanOfFragment(text, "two\nthree\n")
	if !ok || first != 2 || last != 3 {
		t.Errorf("spanOfFragment = %d..%d, %t, want 2..3", first, last, ok)
	}

	if _, _, ok := spanOfFragment(text, "not in the file"); ok {
		t.Error("a fragment the file does not hold was given a span")
	}
	if _, _, ok := spanOfFragment(text, ""); ok {
		t.Error("an empty fragment was given a span")
	}
}
