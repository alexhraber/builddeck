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
				{LineIndex: 0, FilePath: "app/handler.ts", Line: 15, Column: 20, StartCol: 10, EndCol: 30},
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
