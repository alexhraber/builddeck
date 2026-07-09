package tui

import (
	"strings"

	"github.com/alexhraber/builddeck/internal/buildkite"
)

var hardcodedEmoji = map[string]string{
	":buildkite:": "🚀",
	"github":      "🐙",
	"git":         "",
	"docker":      "🐳",
	"go":          "🐹",
	"golang":      "🐹",
	"python":      "🐍",
	"rust":        "🦀",
	"ruby":        "💎",
	"node":        "",
	"npm":         "📦",
	"aws":         "☁️",
	"gcp":         "☁️",
	"azure":       "☁️",
	"linux":       "🐧",
	"apple":       "🍎",
	"windows":     "",
	"test":        "🧪",
	"lint":        "🔍",
	"tidy":        "🧹",
	"check":       "✅",
	"x":           "❌",
	"warning":     "⚠️",
	"rocket":      "🚀",
	"shipit":      "🚀",
	"star":        "⭐",
	"sparkles":    "✨",
	"fire":        "🔥",
	"zap":         "⚡",
	"eyes":        "👀",
	"book":        "📖",
	"hammer":      "🔨",
	"wrench":      "🔧",
	"bug":         "🐛",
	"beers":       "🍺",
	"lock":        "🔒",
	"arrow_up":    "⬆️",
	"arrow_down":  "⬇️",
	"arrow_left":  "⬅️",
	"arrow_right": "➡️",
	"white_check_mark": "✅",
	"red_circle":       "🔴",
	"green_circle":     "🟢",
	"yellow_circle":    "🟡",
	"large_blue_circle": "🔵",
	"heart":   "❤️",
	"100":     "💯",
	"clap":    "👏",
	"tada":    "🎉",
	"package": "📦",
	"art":     "🎨",
	"fast_forward": "⏩",
	"recycle":      "♻️",
	"memo":         "📝",
}

var emojiMap map[string]string

func init() {
	initEmojiMap(nil)
}

func initEmojiMap(apiEmojis []buildkite.EmojiEntry) {
	m := make(map[string]string, len(hardcodedEmoji)+len(apiEmojis))
	for name, glyph := range hardcodedEmoji {
		m[":"+name+":"] = glyph
	}
	for _, e := range apiEmojis {
		key := ":" + e.Name + ":"
		if _, ok := m[key]; !ok {
			m[key] = ""
		}
	}
	emojiMap = m
}

func renderEmoji(s string) string {
	if emojiMap == nil {
		return s
	}
	var buf strings.Builder
	i := 0
	for i < len(s) {
		colon := strings.IndexByte(s[i:], ':')
		if colon < 0 {
			buf.WriteString(s[i:])
			break
		}
		pos := i + colon
		buf.WriteString(s[i:pos])
		end := strings.IndexByte(s[pos+1:], ':')
		if end < 0 {
			buf.WriteString(s[pos:])
			break
		}
		shortcode := s[pos : pos+end+2]
		glyph, ok := emojiMap[shortcode]
		if ok && glyph != "" {
			buf.WriteString(glyph)
		} else if ok {
			buf.WriteString(shortcode[1 : len(shortcode)-1])
		} else {
			buf.WriteString(shortcode)
		}
		i = pos + end + 2
	}
	return buf.String()
}
