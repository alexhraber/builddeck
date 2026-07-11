package tui

import (
	"strings"
	"testing"
)

func TestFormatBuildMessageTruncatesAt40Characters(t *testing.T) {
	message := strings.Repeat("a", maxMsgLen+10)

	got := formatBuildMessage(message)

	if want := strings.Repeat("a", maxMsgLen) + "…"; got != want {
		t.Fatalf("formatBuildMessage() = %q, want %q", got, want)
	}
}
