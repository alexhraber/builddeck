package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// ansiPattern matches ANSI/terminal escape sequences:
	//   CSI: \x1b[31m, \x1b[1m etc.
	//   OSC: \x1b]bk;t=...\x1b\\ (kitty protocol)
	ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x1b]*(?:\x1b\\|\x07)|\x1b.`)

	// refPattern matches common compiler/test file:line patterns:
	//   src/main.go:42
	//   /home/ci/build/src/main.go:42
	//   src/file.ts:10:20
	//   ./relative/path.py:42
	refPattern = regexp.MustCompile(`([^\s]\S*?\.\w+):(\d+)(?::(\d+))?`)

	// altRefPattern matches quoted filename patterns from test frameworks:
	//   File "src/main.py", line 42
	//   at src/app.ts:42:20
	altRefPattern = regexp.MustCompile(`(?:File\s+"|at\s+)(\S+?\.\w+)(?:",\s*line\s+|:)(\d+)(?::(\d+))?`)
)

// stripANSI removes ANSI/terminal escape sequences from s.
func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// parseSourceRefs scans log text and returns all detected file:line references.
func parseSourceRefs(log string) []LogSourceRef {
	if log == "" {
		return nil
	}
	log = stripANSI(log)
	lines := splitLinesPreserveIndex(log)
	seen := make(map[string]bool)
	var refs []LogSourceRef
	for li, line := range lines {
		for _, ref := range extractRefs(line, li, refPattern, 1, 2, 3) {
			key := fmt.Sprintf("%s:%d:%d", ref.FilePath, ref.Line, ref.Column)
			if !seen[key] {
				seen[key] = true
				refs = append(refs, ref)
			}
		}
		for _, ref := range extractRefs(line, li, altRefPattern, 1, 2, 3) {
			key := fmt.Sprintf("%s:%d:%d", ref.FilePath, ref.Line, ref.Column)
			if !seen[key] {
				seen[key] = true
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

func extractRefs(line string, lineIndex int, re *regexp.Regexp, fileGroup, lineGroup, colGroup int) []LogSourceRef {
	matches := re.FindAllStringSubmatchIndex(line, -1)
	var refs []LogSourceRef
	for _, m := range matches {
		if len(m) < (lineGroup+1)*2 {
			continue
		}
		fileStart, fileEnd := m[fileGroup*2], m[fileGroup*2+1]
		lineStart, lineEnd := m[lineGroup*2], m[lineGroup*2+1]
		if fileStart < 0 || fileEnd < 0 || lineStart < 0 || lineEnd < 0 {
			continue
		}
		filePath := strings.TrimLeft(line[fileStart:fileEnd], "/")
		lineNum, err := strconv.Atoi(line[lineStart:lineEnd])
		if err != nil {
			continue
		}
		colNum := 0
		if colGroup >= 0 && len(m) >= (colGroup+1)*2 && m[colGroup*2] >= 0 && m[colGroup*2+1] > m[colGroup*2] {
			colNum, _ = strconv.Atoi(line[m[colGroup*2]:m[colGroup*2+1]])
		}
		refs = append(refs, LogSourceRef{
			LineIndex: lineIndex,
			FilePath:  filePath,
			Line:      lineNum,
			Column:    colNum,
			StartCol:  m[0],
			EndCol:    m[1],
		})
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
