package tui

import (
	"testing"

	"github.com/alexhraber/builddeck/internal/buildkite"
)

func TestInitEmojiBankGo(t *testing.T) {
	emojiMu.RLock()
	entry, ok := emojiBank[":go:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :go: to be in emojiBank after init")
	}
	if entry.glyph == "" {
		t.Fatal("expected :go: to have a glyph")
	}
}

func TestInitEmojiBankGolang(t *testing.T) {
	emojiMu.RLock()
	entry, ok := emojiBank[":golang:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :golang: to be in emojiBank after init")
	}
	if entry.glyph == "" {
		t.Fatal("expected :golang: to have a glyph")
	}
}

func TestInitEmojiBankDocker(t *testing.T) {
	emojiMu.RLock()
	entry, ok := emojiBank[":docker:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :docker: to be in emojiBank after init")
	}
	if entry.glyph == "" {
		t.Fatal("expected :docker: to have a glyph")
	}
}

func TestInitEmojiBankBuildkite(t *testing.T) {
	emojiMu.RLock()
	entry, ok := emojiBank[":buildkite:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :buildkite: to be in emojiBank after init")
	}
	if entry.glyph == "" {
		t.Fatal("expected :buildkite: to have a glyph")
	}
}

func TestRenderEmojiGo(t *testing.T) {
	result := renderEmoji(":go:")
	if result == "" {
		t.Fatal("renderEmoji(:go:) returned empty")
	}
	if result == "go" {
		t.Fatal("renderEmoji(:go:) returned literal 'go', expected glyph")
	}
}

func TestRenderEmojiGolang(t *testing.T) {
	result := renderEmoji(":golang:")
	if result == "" {
		t.Fatal("renderEmoji(:golang:) returned empty")
	}
	if result == "golang" {
		t.Fatal("renderEmoji(:golang:) returned literal 'golang', expected glyph")
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
	result := renderEmoji(":docker: Build")
	if result == "" {
		t.Fatal("renderEmoji(\":docker: Build\") returned empty")
	}
	if result == "docker Build" {
		t.Fatal("renderEmoji(\":docker: Build\") returned literal, expected glyph")
	}
}

func TestRenderEmojiMultiple(t *testing.T) {
	result := renderEmoji(":docker: Build :go:")
	if result == "" {
		t.Fatal("renderEmoji(\":docker: Build :go:\") returned empty")
	}
	if result == "docker Build go" {
		t.Fatal("renderEmoji(\":docker: Build :go:\") returned literal, expected glyphs")
	}
}

func TestRenderEmojiPartialColon(t *testing.T) {
	result := renderEmoji(":go:without closing")
	if result == "" {
		t.Fatal("renderEmoji(\":go:without closing\") returned empty")
	}
	if result == "gowithout closing" {
		t.Fatal("renderEmoji(\":go:without closing\") returned literal, expected glyph")
	}
}

func TestRenderEmojiStrayColon(t *testing.T) {
	result := renderEmoji("text: not emoji")
	if result != "text: not emoji" {
		t.Fatalf("renderEmoji(\"text: not emoji\") = %q, want raw", result)
	}
}

func TestRenderEmojiBankNotNil(t *testing.T) {
	emojiMu.RLock()
	l := len(emojiBank)
	emojiMu.RUnlock()
	if l < 30 {
		t.Fatalf("emojiBank has %d entries, expected at least 30", l)
	}
}

func TestRenderEmojiBroom(t *testing.T) {
	result := renderEmoji(":broom: Tidy")
	if result == ":broom: Tidy" {
		t.Fatal("expected :broom: to be rendered, got raw")
	}
}

func TestRenderEmojiDetective(t *testing.T) {
	result := renderEmoji(":female_detective: Lint")
	if result == ":female_detective: Lint" {
		t.Fatal("expected :female_detective: to be rendered, got raw")
	}
}

func TestRenderEmojiConstruction(t *testing.T) {
	result := renderEmoji(":construction: WIP")
	if result == ":construction: WIP" {
		t.Fatal("expected :construction: to be rendered, got raw")
	}
}

func TestRenderEmojiTestTube(t *testing.T) {
	result := renderEmoji(":test_tube: Tests")
	if result == ":test_tube: Tests" {
		t.Fatal("expected :test_tube: to be rendered, got raw")
	}
}

func TestInitEmojiMap(t *testing.T) {
	origLen := len(emojiBank)
	emojis := []buildkite.EmojiEntry{{Name: "custom_test_emoji_xyz"}}
	initEmojiMap(emojis)
	emojiMu.RLock()
	_, ok := emojiBank[":custom_test_emoji_xyz:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :custom_test_emoji_xyz: to be in emojiBank after initEmojiMap")
	}
	emojiMu.RLock()
	newLen := len(emojiBank)
	emojiMu.RUnlock()
	if newLen != origLen+1 {
		t.Fatalf("emojiBank grew by %d (expected 1)", newLen-origLen)
	}
}
