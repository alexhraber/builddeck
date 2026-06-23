package tui

import (
	"fmt"
	"time"
)

func FormatTime(s string) string {
	if s == "" {
		return "—"
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("Jan 02 15:04")
}

func FormatDuration(start, finish string) string {
	if start == "" {
		return "—"
	}
	s, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return "—"
	}
	var f time.Time
	if finish != "" {
		f, err = time.Parse(time.RFC3339, finish)
		if err != nil {
			return "—"
		}
	} else {
		f = time.Now()
	}
	d := f.Sub(s)
	if d < 0 {
		return "—"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
