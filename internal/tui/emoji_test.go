package tui

import (
	"strings"
	"testing"
)

func TestInitEmojiBank(t *testing.T) {
	emojiMu.RLock()
	entry, ok := emojiBank[":go:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :go: to be in emojiBank after init")
	}
	if entry.assetGlyph == "" {
		t.Fatal("expected :go: to have assetGlyph from golang alias")
	}
}

func TestInitEmojiBankDocker(t *testing.T) {
	emojiMu.RLock()
	entry, ok := emojiBank[":docker:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :docker: to be in emojiBank after init")
	}
	if entry.assetGlyph == "" {
		t.Fatal("expected :docker: to have assetGlyph from docker.png")
	}
}

func TestInitEmojiBankBuildkite(t *testing.T) {
	emojiMu.RLock()
	_, ok := emojiBank[":buildkite:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :buildkite: to be in emojiBank after init")
	}
}

func TestRenderEmojiKnown(t *testing.T) {
	result := renderEmoji(":go:")
	if result != "go" {
		t.Fatalf("renderEmoji(:go:) = %q, want shortcode name", result)
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
	expected := "go Tidy"
	if result != expected {
		t.Fatalf("renderEmoji(\":go: Tidy\") = %q, want %q", result, expected)
	}
}

func TestRenderEmojiMultiple(t *testing.T) {
	result := renderEmoji(":docker: Build :go:")
	expected := "docker Build go"
	if result != expected {
		t.Fatalf("renderEmoji(\":docker: Build :go:\") = %q, want %q", result, expected)
	}
}

func TestRenderEmojiPartialColon(t *testing.T) {
	result := renderEmoji(":go:without closing")
	want := "gowithout closing"
	if result != want {
		t.Fatalf("renderEmoji(\":go:without closing\") = %q, want %q", result, want)
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

func TestPipelineEmojiBank(t *testing.T) {
	emojiMu.RLock()
	entry, ok := emojiBank[":buildkite:"]
	emojiMu.RUnlock()
	if !ok {
		t.Fatal("expected :buildkite: to be in emojiBank")
	}
	if entry.glyph == "" && entry.assetGlyph == "" {
		t.Fatal("expected :buildkite: to have a glyph or assetGlyph")
	}
}

func TestLoadPipelineEmoji(t *testing.T) {
	result := loadPipelineEmoji("buildkite")
	if result == "" {
		t.Fatal("loadPipelineEmoji(\"buildkite\") returned empty")
	}
	if result == ":buildkite:" {
		t.Fatal("loadPipelineEmoji returned raw fallback, expected rendered glyph")
	}
}

func TestLoadPipelineEmojiMissing(t *testing.T) {
	result := loadPipelineEmoji("clearly-nonexistent-emoji-name")
	if result == "" {
		t.Fatal("loadPipelineEmoji missing should fall back, not be empty")
	}
}

func TestLoadPipelineEmojiGolang(t *testing.T) {
	result := loadPipelineEmoji("golang")
	if result == "" {
		t.Fatal("loadPipelineEmoji(\"golang\") returned empty")
	}
	if strings.HasPrefix(result, ":") {
		t.Fatalf("loadPipelineEmoji(\"golang\") got raw shortcode %q", result)
	}
}

func TestLoadPipelineEmojiPipeline(t *testing.T) {
	result := loadPipelineEmoji("pipeline")
	if result == "" {
		t.Fatal("loadPipelineEmoji(\"pipeline\") returned empty")
	}
	if strings.HasPrefix(result, ":") {
		t.Fatalf("loadPipelineEmoji(\"pipeline\") got raw shortcode %q", result)
	}
}

func TestLoadPipelineEmojiGo(t *testing.T) {
	result := loadPipelineEmoji("go")
	if result == "" {
		t.Fatal("loadPipelineEmoji(\"go\") returned empty")
	}
	if strings.HasPrefix(result, ":") {
		t.Fatalf("loadPipelineEmoji(\"go\") got raw shortcode %q", result)
	}
	// go should use golang.png as alias
	if result == "go" {
		t.Fatal("loadPipelineEmoji(\"go\") returned literal 'go', expected golang asset glyph")
	}
}

func TestLoadPipelineEmojiBuildkiteParty(t *testing.T) {
	result := loadPipelineEmoji("buildkite-party")
	if result == "" {
		t.Fatal("loadPipelineEmoji(\"buildkite-party\") returned empty")
	}
	if strings.HasPrefix(result, ":") {
		t.Fatalf("loadPipelineEmoji(\"buildkite-party\") got raw shortcode %q", result)
	}
}

func TestLoadPipelineEmojiGithub(t *testing.T) {
	result := loadPipelineEmoji("github")
	if result == "" {
		t.Fatal("loadPipelineEmoji(\"github\") returned empty")
	}
	if strings.HasPrefix(result, ":") {
		t.Fatalf("loadPipelineEmoji(\"github\") got raw shortcode %q", result)
	}
}
