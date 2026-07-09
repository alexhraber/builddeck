package tui

import (
	"testing"
)

func TestInitEmojiMap(t *testing.T) {
	emojiMu.RLock()
	glyph, ok := emojiMap[":go:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :go: to be in emojiMap after init")
	}
	if glyph != "\ue627" {
		t.Fatalf("expected :go: glyph to be \\ue627, got %q", glyph)
	}
}

func TestInitEmojiMapDocker(t *testing.T) {
	emojiMu.RLock()
	glyph, ok := emojiMap[":docker:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :docker: to be in emojiMap after init")
	}
	if glyph != "\uf21b" {
		t.Fatalf("expected :docker: glyph to be \\uf21b, got %q", glyph)
	}
}

func TestInitEmojiMapBuildkite(t *testing.T) {
	emojiMu.RLock()
	_, ok := emojiMap[":buildkite:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :buildkite: to be in emojiMap after init")
	}
}

func TestRenderEmojiKnown(t *testing.T) {
	result := renderEmoji(":go:")
	if result != "\ue627" {
		t.Fatalf("renderEmoji(:go:) = %q, want \\ue627", result)
	}
}

func TestRenderEmojiUnknown(t *testing.T) {
	result := renderEmoji(":nonexistent_emoji:")
	if result != ":nonexistent_emoji:" {
		t.Fatalf("renderEmoji(:nonexistent_emoji:) = %q, want raw string", result)
	}
}

func TestRenderEmojiEmpty(t *testing.T) {
	result := renderEmoji("")
	if result != "" {
		t.Fatalf("renderEmoji(\"\") = %q, want empty", result)
	}
}

func TestRenderEmojiNoShortcodes(t *testing.T) {
	input := "hello world"
	result := renderEmoji(input)
	if result != input {
		t.Fatalf("renderEmoji(%q) = %q, want %q", input, result, input)
	}
}

func TestRenderEmojiInContext(t *testing.T) {
	result := renderEmoji(":go: Tidy")
	expected := "\ue627 Tidy"
	if result != expected {
		t.Fatalf("renderEmoji(\":go: Tidy\") = %q, want %q", result, expected)
	}
}

func TestRenderEmojiMultiple(t *testing.T) {
	result := renderEmoji(":docker: Build :go:")
	expected := "\uf21b Build \ue627"
	if result != expected {
		t.Fatalf("renderEmoji(\":docker: Build :go:\") = %q, want %q", result, expected)
	}
}

func TestRenderEmojiPartialColon(t *testing.T) {
	// ":go:" is found as a valid shortcode, then "without closing" has
	// no more colons and passes through as-is.
	result := renderEmoji(":go:without closing")
	want := "\ue627without closing"
	if result != want {
		t.Fatalf("renderEmoji(\":go:without closing\") = %q, want %q", result, want)
	}
}

func TestRenderEmojiStrayColon(t *testing.T) {
	// A single colon that is not part of a :shortcode: passes through.
	result := renderEmoji("text: not emoji")
	if result != "text: not emoji" {
		t.Fatalf("renderEmoji(\"text: not emoji\") = %q, want raw", result)
	}
}

func TestRenderEmojiMapNotNil(t *testing.T) {
	emojiMu.RLock()
	l := len(emojiMap)
	emojiMu.RUnlock()
	if l < 30 {
		t.Fatalf("emojiMap has %d entries, expected at least 30", l)
	}
}
