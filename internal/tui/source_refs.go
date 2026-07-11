package tui

import (
	"regexp"
	"strconv"
)

var (
	// refPattern matches common compiler/test file:line patterns:
	//   path/to/file.go:42
	//   path/to/file.ts:10:20  (with column)
	//   ./relative/path.py:42
	//   /absolute/path.rs:42:5
	refPattern = regexp.MustCompile(`([^/\s]\S*?\.\w+):(\d+)(?::(\d+))?`)

	// goErrorPattern matches Go compiler error continuation lines:
	//   /path/file.go:42:21: undefined: Foo
	// (captures the file:line:col part, same as above)
)

// parseSourceRefs scans log text and returns all detected file:line references.
func parseSourceRefs(log string) []LogSourceRef {
	if log == "" {
		return nil
	}
	lines := splitLinesPreserveIndex(log)
	var refs []LogSourceRef
	for li, line := range lines {
		matches := refPattern.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			if len(m) < 6 {
				continue
			}
			fullStart := m[0]
			// fullEnd := m[1]
			fileStart := m[2]
			fileEnd := m[3]
			lineStart := m[4]
			lineEnd := m[5]

			filePath := line[fileStart:fileEnd]
			lineNum, err := strconv.Atoi(line[lineStart:lineEnd])
			if err != nil {
				continue
			}

			colNum := 0
			if len(m) > 6 && m[6] >= 0 && m[7] > m[6] {
				colNum, _ = strconv.Atoi(line[m[6]:m[7]])
			}

			refs = append(refs, LogSourceRef{
				LineIndex: li,
				FilePath:  filePath,
				Line:      lineNum,
				Column:    colNum,
				StartCol:  fullStart,
				EndCol:    m[1],
			})
		}
	}
	return refs
}

// splitLinesPreserveIndex splits text by newlines and returns each line
// with its original byte-offset index preserved via the slice index.
func splitLinesPreserveIndex(log string) []string {
	if log == "" {
		return nil
	}
	n := len(log)
	var lines []string
	start := 0
	for i := 0; i < n; i++ {
		if log[i] == '\n' {
			lines = append(lines, log[start:i])
			start = i + 1
		}
	}
	if start < n {
		lines = append(lines, log[start:])
	} else if start == n && n > 0 && log[n-1] == '\n' {
		lines = append(lines, "")
	}
	return lines
}
