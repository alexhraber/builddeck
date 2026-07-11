package tui

import (
	"testing"
)

func TestParseSourceRefs(t *testing.T) {
	tests := []struct {
		name string
		log  string
		want []LogSourceRef
	}{
		{
			name: "empty log",
			log:  "",
			want: nil,
		},
		{
			name: "no refs",
			log:  "Build started\nBuild finished",
			want: nil,
		},
		{
			name: "go compiler error",
			log:  "# github.com/example/pkg\nsrc/main.go:42: undefined: Foo\nsrc/utils.go:10:6: too many arguments",
			want: []LogSourceRef{
				{LineIndex: 1, FilePath: "src/main.go", Line: 42, Column: 0, StartCol: 0, EndCol: 14},
				{LineIndex: 2, FilePath: "src/utils.go", Line: 10, Column: 6, StartCol: 0, EndCol: 17},
			},
		},
		{
			name: "with column",
			log:  "Error in /app/handler.ts:15:20",
			want: []LogSourceRef{
				{LineIndex: 0, FilePath: "app/handler.ts", Line: 15, Column: 20, StartCol: 9, EndCol: 30},
			},
		},
		{
			name: "test framework output",
			log:  "=== FAIL: TestSomething\n    testing.go:123: assertion failed\n        some_test.go:45: expected true, got false",
			want: []LogSourceRef{
				{LineIndex: 1, FilePath: "testing.go", Line: 123, Column: 0, StartCol: 4, EndCol: 18},
				{LineIndex: 2, FilePath: "some_test.go", Line: 45, Column: 0, StartCol: 8, EndCol: 23},
			},
		},
		{
			name: "absolute path at line start",
			log:  "/home/ci/build/src/main.go:42: undefined",
			want: []LogSourceRef{
				{LineIndex: 0, FilePath: "home/ci/build/src/main.go", Line: 42, Column: 0, StartCol: 0, EndCol: 29},
			},
		},
		{
			name: "no extension no match",
			log:  "Makefile:42: error",
			want: nil,
		},
		{
			name: "Python File pattern",
			log:  "  File \"src/main.py\", line 42, in foo",
			want: []LogSourceRef{
				{LineIndex: 0, FilePath: "src/main.py", Line: 42, Column: 0, StartCol: 2, EndCol: 29},
			},
		},
		{
			name: "JavaScript at pattern",
			log:  "at src/app.ts:42:20",
			want: []LogSourceRef{
				{LineIndex: 0, FilePath: "src/app.ts", Line: 42, Column: 20, StartCol: 3, EndCol: 19},
			},
		},
		{
			name: "multiple patterns in one line",
			log:  "src/a.go:10 and src/b.go:20:30",
			want: []LogSourceRef{
				{LineIndex: 0, FilePath: "src/a.go", Line: 10, Column: 0, StartCol: 0, EndCol: 11},
				{LineIndex: 0, FilePath: "src/b.go", Line: 20, Column: 30, StartCol: 16, EndCol: 30},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSourceRefs(tt.log)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d refs, want %d\ngot: %+v", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i].LineIndex != tt.want[i].LineIndex {
					t.Errorf("ref[%d] LineIndex = %d, want %d", i, got[i].LineIndex, tt.want[i].LineIndex)
				}
				if got[i].FilePath != tt.want[i].FilePath {
					t.Errorf("ref[%d] FilePath = %q, want %q", i, got[i].FilePath, tt.want[i].FilePath)
				}
				if got[i].Line != tt.want[i].Line {
					t.Errorf("ref[%d] Line = %d, want %d", i, got[i].Line, tt.want[i].Line)
				}
				if got[i].Column != tt.want[i].Column {
					t.Errorf("ref[%d] Column = %d, want %d", i, got[i].Column, tt.want[i].Column)
				}
				if got[i].StartCol != tt.want[i].StartCol {
					t.Errorf("ref[%d] StartCol = %d, want %d", i, got[i].StartCol, tt.want[i].StartCol)
				}
				if got[i].EndCol != tt.want[i].EndCol {
					t.Errorf("ref[%d] EndCol = %d, want %d", i, got[i].EndCol, tt.want[i].EndCol)
				}
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"no ansi", "hello world", "hello world"},
		{"CSI SGR", "\x1b[31mred\x1b[0m", "red"},
		{"OSC kitty", "\x1b]bk;t=123\x07path", "path"},
		{"OSC with ST", "\x1b]bk;t=123\x1b\\path", "path"},
		{"mixed", "\x1b[1m\x1b[31m  \x1b[0mhello.go:42", "  hello.go:42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.in)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseSourceRefsWithANSI(t *testing.T) {
	log := "\x1b[1m\x1b[31m  \x1b[0m\x1b]bk;t=123\x07src/main.go:42: error"
	refs := parseSourceRefs(log)
	if len(refs) != 1 {
		t.Fatalf("got %d refs, want 1\nrefs: %+v", len(refs), refs)
	}
	if refs[0].FilePath != "src/main.go" {
		t.Errorf("FilePath = %q, want src/main.go", refs[0].FilePath)
	}
	if refs[0].Line != 42 {
		t.Errorf("Line = %d, want 42", refs[0].Line)
	}
}

func TestSplitLinesPreserveIndex(t *testing.T) {
	tests := []struct {
		name string
		log  string
		want int
	}{
		{"empty", "", 0},
		{"single line", "hello", 1},
		{"multiple lines", "a\nb\nc", 3},
		{"trailing newline", "a\nb\n", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLinesPreserveIndex(tt.log)
			if len(got) != tt.want {
				t.Errorf("got %d lines, want %d: %q", len(got), tt.want, got)
			}
		})
	}
}

func TestSetLogContent(t *testing.T) {
	m := NewModel(nil)
	m.setLogContent("src/main.go:42: some error")
	if m.currentLog != "src/main.go:42: some error" {
		t.Errorf("currentLog = %q", m.currentLog)
	}
	if len(m.logSourceRefs) != 1 {
		t.Fatalf("expected 1 source ref, got %d", len(m.logSourceRefs))
	}
	if m.logSourceRefs[0].FilePath != "src/main.go" {
		t.Errorf("FilePath = %q", m.logSourceRefs[0].FilePath)
	}
	if m.logSourceRefs[0].Line != 42 {
		t.Errorf("Line = %d", m.logSourceRefs[0].Line)
	}
	if m.logSourceIndex != 0 {
		t.Errorf("logSourceIndex = %d", m.logSourceIndex)
	}

	m.setLogContent("no refs here")
	if len(m.logSourceRefs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(m.logSourceRefs))
	}
	if m.logSourceIndex != -1 {
		t.Errorf("logSourceIndex = %d, want -1", m.logSourceIndex)
	}
}
