package tui

import (
	"image"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alexhraber/builddeck/internal/buildkite"
	"github.com/charmbracelet/lipgloss"
)

var (
	emojiMap   map[string]string
	emojiMu    sync.RWMutex
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

// nerdFontIcons maps Buildkite emoji names to Nerd Font codepoints.
// Users with Nerd Fonts installed will see crisp icons instead of block art.
var nerdFontIcons = map[string]string{
	"docker":       "\uf21b", //  fa-docker
	"go":           "\ue627", //  dev-go
	"golang":       "\ue627", //  dev-go
	"python":       "\ue73c", //  dev-python
	"rust":         "\ue7a8", // 
	"ruby":         "\ue791", // 
	"node":         "\ue718", //  dev-nodejs_small
	"npm":          "\ue71e", //  dev-npm
	"github":       "\uf09b", //  fa-github
	"git":          "\uf1d3", //  fa-git-alt
	"aws":          "\uf0c2", //  fa-cloud
	"gcp":          "\uf0c2", //  fa-cloud
	"azure":        "\uf0c2", //  fa-cloud
	"linux":        "\uf17c", //  fa-linux
	"apple":        "\uf179", //  fa-apple
	"windows":      "\uf17a", //  fa-windows
	"test":         "\uf478", //  oct-check
	"check":        "\uf00c", //  fa-check
	"star":         "\uf005", //  fa-star
	"heart":        "\uf004", //  fa-heart
	"bug":          "\uf188", //  fa-bug
	"rocket":       "\uf135", //  fa-rocket
	"shipit":       "\uf135", //  fa-rocket
	"warning":      "\uf071", //  fa-warning
	"lock":         "\uf023", //  fa-lock
	"fire":         "\uf06d", //  fa-fire
	"tada":         "\uf0e7", //  fa-bolt (zap)
	"package":      "\uf187", //  fa-archive
	"art":          "\uf1fc", //  fa-paint-brush
	"book":         "\uf02d", //  fa-book
	"hammer":       "\uf0e7", //  fa-bolt
	"fast_forward": "\u23e9", // ⏩
}

func init() {
	emojiMap = make(map[string]string)
}

func initEmojiMap(apiEmojis []buildkite.EmojiEntry) {
	emojiMu.Lock()
	defer emojiMu.Unlock()

	// seed with nerd font icons
	for name, glyph := range nerdFontIcons {
		emojiMap[":"+name+":"] = glyph
	}

	for _, e := range apiEmojis {
		key := ":" + e.Name + ":"
		if _, exists := emojiMap[key]; !exists {
			emojiMap[key] = ""
		}
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for _, e := range apiEmojis {
		key := ":" + e.Name + ":"
		emojiMu.RLock()
		already := emojiMap[key]
		emojiMu.RUnlock()
		if already != "" || e.URL == "" {
			continue
		}
		wg.Add(1)
		go func(entry buildkite.EmojiEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			glyph := downloadEmoji(entry.URL)
			if glyph != "" {
				emojiMu.Lock()
				emojiMap[":"+entry.Name+":"] = glyph
				emojiMu.Unlock()
			}
		}(e)
	}
	wg.Wait()
}

func downloadEmoji(url string) string {
	resp, err := httpClient.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	limited := io.LimitReader(resp.Body, 256*1024)
	img, _, err := image.Decode(limited)
	if err != nil {
		return ""
	}
	return renderEmojiGlyph(img)
}

func renderEmojiGlyph(img image.Image) string {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w == 0 || h == 0 {
		return ""
	}

	sample := func(x, y int) (uint8, uint8, uint8, uint8) {
		sx := bounds.Min.X + x*w/w
		sy := bounds.Min.Y + y*h/h
		r, g, b, a := img.At(sx, sy).RGBA()
		return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)
	}

	gridW := 2
	gridH := 2

	pixels := make([][4]uint8, gridW*gridH)
	for y := range gridH {
		for x := range gridW {
			r, g, b, a := sample(x, y)
			if a < 128 {
				pixels[y*gridW+x] = [4]uint8{0, 0, 0, 0}
			} else {
				pixels[y*gridW+x] = [4]uint8{r, g, b, a}
			}
		}
	}

	nearest := func(c [4]uint8) lipgloss.Color {
		return lipgloss.Color(uint8ToHex(c[0], c[1], c[2]))
	}

	var buf strings.Builder
	for x := range gridW {
		top := pixels[x]
		bottom := pixels[(gridH-1)*gridW+x]
		if top[3] == 0 && bottom[3] == 0 {
			buf.WriteString(" ")
			continue
		}
		fg := nearest(top)
		bg := nearest(bottom)
		char := "▀"
		if top[3] == 0 {
			fg = bg
			char = " "
		}
		s := lipgloss.NewStyle().
			Foreground(fg).
			Background(bg).
			Render(char)
		buf.WriteString(s)
	}
	return buf.String()
}

func uint8ToHex(r, g, b uint8) string {
	hex := [7]byte{'#', 0, 0, 0, 0, 0, 0}
	const hextable = "0123456789abcdef"
	hex[1] = hextable[r>>4]
	hex[2] = hextable[r&0x0f]
	hex[3] = hextable[g>>4]
	hex[4] = hextable[g&0x0f]
	hex[5] = hextable[b>>4]
	hex[6] = hextable[b&0x0f]
	return string(hex[:])
}

func renderEmoji(s string) string {
	emojiMu.RLock()
	defer emojiMu.RUnlock()

	if len(s) == 0 {
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
		if ok {
			if glyph != "" {
				buf.WriteString(glyph)
			} else {
				buf.WriteString(shortcode[1 : len(shortcode)-1])
			}
		} else {
			buf.WriteString(shortcode)
		}
		i = pos + end + 2
	}
	return buf.String()
}
