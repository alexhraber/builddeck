package tui

import (
	"embed"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/png"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alexhraber/builddeck/internal/buildkite"
	"github.com/charmbracelet/lipgloss"
)

//go:embed emoji_assets
var emojiAssets embed.FS

const emojiAssetsPrefix = "emoji_assets"
const gridW = 2
const gridH = 2

type emojiEntry struct {
	glyph      string // Nerd Font / Unicode for inline use
	assetGlyph string // 2-cell half-block art from PNG/GIF
}

	var (
	emojiBank     map[string]emojiEntry
	emojiMu       sync.RWMutex
	httpClient    = &http.Client{Timeout: 10 * time.Second}
	assetAliases  = map[string]string{"go": "golang"}
)

// nerdFontIcons maps Buildkite emoji names to Nerd Font codepoints.
// Users with Nerd Fonts installed will see crisp icons instead of block art.
var nerdFontIcons = map[string]string{
	"go":              "\ue627",  //  dev-go
	"gcp":             "\uf0c2",  //  fa-cloud
	"apple":           "\uf179",  //  fa-apple
	"test":            "\uf478",  //  oct-check
	"check":           "\uf00c",  //  fa-check
	"star":            "\uf005",  //  fa-star
	"heart":           "\uf004",  //  fa-heart
	"bug":             "\uf188",  //  fa-bug
	"rocket":          "\uf135",  //  fa-rocket
	"warning":         "\uf071",  //  fa-warning
	"lock":            "\uf023",  //  fa-lock
	"fire":            "\uf06d",  //  fa-fire
	"tada":            "\uf0e7",  //  fa-bolt (zap)
	"package":         "\uf187",  //  fa-archive
	"art":             "\uf1fc",  //  fa-paint-brush
	"book":            "\uf02d",  //  fa-book
	"hammer":          "\uf0e7",  //  fa-bolt
	"fast_forward":    "\u23e9",  // ⏩
	"sparkles":        "\uf00a",  //  md-sparkles
	"merge":           "\uf157",  //  fa-code-fork
	"zap":             "\uf0e7",  //  fa-bolt
	"lightning":       "\uf0e7",  //  fa-bolt
	"gear":            "\uf013",  //  fa-gear
	"cog":             "\uf013",  //  fa-cog
	"wrench":          "\uf0ad",  //  fa-wrench
	"tools":           "\uf0ad",  //  fa-wrench
	"building":        "\uf1ad",  //  fa-building
	"office":          "\uf1ad",  //  fa-building
	"home":            "\uf015",  //  fa-home
	"seedling":        "\uf4d8",  //  oct-seedling
	"play":            "\uf04b",  //  fa-play
	"stop":            "\uf04d",  //  fa-stop
	"pause":           "\uf04c",  //  fa-pause
	"sync":            "\uf021",  //  fa-sync
	"refresh":         "\uf021",  //  fa-sync
	"search":          "\uf002",  //  fa-search
	"plus":            "\uf067",  //  fa-plus
	"tag":             "\uf02b",  //  fa-tag
	"branch":          "\uf126",  //  fa-code-fork
	"commit":          "\uf01c",  //  fa-asterisk (or md-source-commit)
	"pr":              "\uf09b",  //  md-source-pull-request
	"pull_request":    "\uf09b",  //  md-source-pull-request
	"docker_compose":  "\uf21b",  //  fa-docker
	"postgresql":      "\ue76e",  //  dev-postgres
	"mongo":           "\ue7a4",  //  dev-mongodb
	"nodejs":          "\ue718",  //  dev-nodejs_small
	"ts":              "\ue628",  //  dev-typescript
	"js":              "\ue781",  //  dev-javascript
	"cplusplus":       "\ue61d",  //  dev-cpp
	"k8s":             "\uf10b",  //  md-kubernetes
	"tf":              "\ue60b",  //  dev-terraform
	"travis":          "\ue77e",  //  dev-travis
	"bitbucket":       "\ue703",  //  dev-bitbucket
	"email":           "\uf0e0",  //  fa-envelope
	"mail":            "\uf0e0",  //  fa-envelope
	"chat":            "\uf075",  //  fa-comment
	"phone":           "\uf095",  //  fa-phone
	"camera":          "\uf030",  //  fa-camera
	"video":           "\uf03d",  //  fa-video-camera
	"youtube":         "\uf167",  //  fa-youtube
	"twitter":         "\uf099",  //  fa-twitter
	"linkedin":        "\uf0e1",  //  fa-linkedin
	"clock":           "\uf017",  //  fa-clock-o
	"time":            "\uf017",  //  fa-clock-o
	"calendar":        "\uf073",  //  fa-calendar
	"date":            "\uf073",  //  fa-calendar
	"eye":             "\uf06e",  //  fa-eye
	"graph":           "\uf080",  //  fa-bar-chart
	"chart":           "\uf080",  //  fa-bar-chart
	"download":        "\uf019",  //  fa-download
	"upload":          "\uf093",  //  fa-upload
	"link":            "\uf0c1",  //  fa-link
	"url":             "\uf0c1",  //  fa-link
	"globe":           "\uf0ac",  //  fa-globe
	"world":           "\uf0ac",  //  fa-globe
	"flag":            "\uf024",  //  fa-flag
	"trophy":          "\uf091",  //  fa-trophy
	"medal":           "\uf0a3",  //  fa-medal
	"award":           "\uf0a3",  //  fa-medal
	"thumbsup":        "\uf164",  //  fa-thumbs-up
	"thumbsdown":      "\uf165",  //  fa-thumbs-down
	"handshake":       "\uf2b5",  //  fa-handshake-o
	"users":           "\uf0c0",  //  fa-users
	"person":          "\uf007",  //  fa-user
	"document":        "\uf016",  //  fa-file-o
	"file":            "\uf016",  //  fa-file-o
	"folder":          "\uf07b",  //  fa-folder
	"archive":         "\uf187",  //  fa-archive
	"zip":             "\uf187",  //  fa-archive
	"key":             "\uf084",  //  fa-key
	"secret":          "\uf084",  //  fa-key
	"shirt":           "\uf553",  //  fa-tshirt
	"ladybug":         "\uf188",  //  fa-bug
	"beetle":          "\uf188",  //  fa-bug
	"ant":             "\uf188",  //  fa-bug
	"snail":           "\uf188",  //  fa-bug
	"turtle":          "\uf188",  //  fa-bug
	"dog":             "\uf6d3",  //  fa-dog
	"cat":             "\uf6be",  //  fa-cat
	"whale":           "\uf72c",  //  fa-whale (Docker!)
	"unicorn":         "\uf15b",  //  fa-html5 (no unicorn in FA)
	"robot":           "\uf544",  //  fa-robot
	"alien":           "\uf8df",  //  fa-alien

	// Standard Unicode emoji commonly used in Buildkite pipeline labels.
	// These are NOT custom org emoji — they're built into Buildkite's web UI,
	// so the API never returns them. We seed them as real Unicode emoji.
	// Both underscore and hyphen variants are provided since Buildkite shortcodes
	// use hyphens (e.g. :female-detective:), while our internal naming uses underscores.
	"broom":                 "\U0001F9F9",   // 🧹
	"detective":             "\U0001F575\uFE0F",           // 🕵️
	"male_detective":        "\U0001F575\uFE0F\u200D\u2642\uFE0F", // 🕵️‍♂️
	"male-detective":        "\U0001F575\uFE0F\u200D\u2642\uFE0F", // 🕵️‍♂️
	"female_detective":      "\U0001F575\uFE0F\u200D\u2640\uFE0F", // 🕵️‍♀️
	"female-detective":      "\U0001F575\uFE0F\u200D\u2640\uFE0F", // 🕵️‍♀️
	"building_construction": "\U0001F3D7", // 🏗
	"building-construction": "\U0001F3D7", // 🏗
	"construction":          "\U0001F6A7", // 🚧
	"test_tube":             "\U0001F9EA", // 🧪
	"test-tube":             "\U0001F9EA", // 🧪
	"white_check_mark":      "\u2705",     // ✅
	"white-check-mark":      "\u2705",     // ✅
	"x":                     "\u274C",     // ❌
	"cross_mark":            "\u274C",     // ❌
	"green_heart":           "\U0001F49A", // 💚
	"green-heart":           "\U0001F49A", // 💚
	"boom":                  "\U0001F4A5", // 💥
	"collision":             "\U0001F4A5", // 💥
	"recycle":               "\u267B\uFE0F", // ♻️
	"pencil":                "\u270F\uFE0F", // ✏️
	"memo":                  "\U0001F4DD", // 📝
	"books":                 "\U0001F4DA", // 📚
	"arrow_up":              "\u2B06",     // ⬆
	"arrow-up":              "\u2B06",     // ⬆
	"arrow_down":            "\u2B07",     // ⬇
	"arrow-down":            "\u2B07",     // ⬇
	"arrow_up_down":         "\u2195",     // ↕
	"heavy_plus_sign":       "\u2795",     // ➕
	"heavy-plus-sign":       "\u2795",     // ➕
	"heavy_minus_sign":      "\u2796",     // ➖
	"heavy-minus-sign":      "\u2796",     // ➖
	"heavy_check_mark":      "\u2714",     // ✔
	"heavy-check-mark":      "\u2714",     // ✔
	"wave":                  "\U0001F44B", // 👋
	"beers":                 "\U0001F37B", // 🍻
	"sweat_smile":           "\U0001F605", // 😅
	"sweat-smile":           "\U0001F605", // 😅
	"smile":                 "\U0001F604", // 😄
	"sob":                   "\U0001F62D", // 😭
	"scream":                "\U0001F631", // 😱
	"buildkite_party":       "\U0001F973", // 🥳 (alias for Buildkite's :buildkite-party:)
	"partying_face":         "\U0001F973", // 🥳
	"partying-face":         "\U0001F973", // 🥳
	"facepalm":              "\U0001F926", // 🤦
}

func init() {
	assets := loadAssetNames()
	emojiBank = make(map[string]emojiEntry, len(nerdFontIcons)+200)
	for name, glyph := range nerdFontIcons {
		if assets[name] {
			continue
		}
		emojiBank[":"+name+":"] = emojiEntry{glyph: glyph}
	}
	loadUnicodeEmojiMap()
	loadAssetEmoji()
}

func loadAssetNames() map[string]bool {
	entries, err := fs.ReadDir(emojiAssets, emojiAssetsPrefix)
	if err != nil {
		return nil
	}
	names := make(map[string]bool, len(entries))
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".png") {
			names[name[:len(name)-4]] = true
		} else if strings.HasSuffix(name, ".gif") {
			names[name[:len(name)-4]] = true
		}
	}
	return names
}

func loadUnicodeEmojiMap() {
	f, err := emojiAssets.Open(emojiAssetsPrefix + "/emoji-unicode.json")
	if err != nil {
		return
	}
	defer f.Close()

	var entries map[string]string
	if err := json.NewDecoder(f).Decode(&entries); err != nil {
		return
	}

	emojiMu.Lock()
	defer emojiMu.Unlock()
	for name, glyph := range entries {
		key := ":" + name + ":"
		if _, exists := emojiBank[key]; !exists {
			emojiBank[key] = emojiEntry{glyph: glyph}
		}
	}
}

func initEmojiMap(apiEmojis []buildkite.EmojiEntry) {
	emojiMu.Lock()
	for _, e := range apiEmojis {
		key := ":" + e.Name + ":"
		if _, exists := emojiBank[key]; !exists {
			emojiBank[key] = emojiEntry{}
		}
	}
	emojiMu.Unlock()

	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for _, e := range apiEmojis {
		key := ":" + e.Name + ":"
		emojiMu.RLock()
		entry := emojiBank[key]
		emojiMu.RUnlock()
		if entry.glyph != "" || entry.assetGlyph != "" || e.URL == "" {
			continue
		}
		wg.Add(1)
		go func(entry buildkite.EmojiEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			glyph := downloadEmoji(entry.URL)
			if glyph != "" {
				key := ":" + entry.Name + ":"
				emojiMu.Lock()
				e := emojiBank[key]
				e.glyph = glyph
				emojiBank[key] = e
				emojiMu.Unlock()
			}
		}(e)
	}
	wg.Wait()
}

func loadAssetEmoji() {
	entries, err := fs.ReadDir(emojiAssets, emojiAssetsPrefix)
	if err != nil {
		return
	}

	emojiMu.Lock()
	defer emojiMu.Unlock()

	for _, entry := range entries {
		name := entry.Name()
		var base string
		if strings.HasSuffix(name, ".png") {
			base = name[:len(name)-4]
		} else if strings.HasSuffix(name, ".gif") {
			base = name[:len(name)-4]
		} else {
			continue
		}

		f, err := emojiAssets.Open(emojiAssetsPrefix + "/" + name)
		if err != nil {
			continue
		}
		img, _, err := image.Decode(io.LimitReader(f, 512*1024))
		f.Close()
		if err != nil {
			continue
		}

		glyph := renderEmojiGlyph(img)
		key := ":" + base + ":"
		if existing, ok := emojiBank[key]; ok {
			existing.assetGlyph = glyph
			emojiBank[key] = existing
		} else {
			emojiBank[key] = emojiEntry{assetGlyph: glyph}
		}
	}
	for alias, canonical := range assetAliases {
		aliasKey := ":" + alias + ":"
		canonicalKey := ":" + canonical + ":"
		if ce, ok := emojiBank[canonicalKey]; ok && ce.assetGlyph != "" {
			if ae, exists := emojiBank[aliasKey]; !exists || ae.assetGlyph == "" {
				emojiBank[aliasKey] = emojiEntry{assetGlyph: ce.assetGlyph}
			}
		}
	}
}

func loadPipelineEmoji(name string) string {
	if name == "" {
		name = "buildkite"
	}

	emojiMu.RLock()
	entry, ok := emojiBank[":"+name+":"]
	emojiMu.RUnlock()

	if ok && entry.assetGlyph != "" {
		return entry.assetGlyph
	}
	if ok && entry.glyph != "" && !isPUA(entry.glyph) {
		return entry.glyph
	}
	return name
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

	type accum struct {
		r, g, b, a, n uint32
	}
	cells := make([]accum, gridW*gridH)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			cellX := (x - bounds.Min.X) * gridW / w
			cellY := (y - bounds.Min.Y) * gridH / h
			idx := cellY*gridW + cellX
			r, g, b, a := img.At(x, y).RGBA()
			cells[idx].r += r
			cells[idx].g += g
			cells[idx].b += b
			cells[idx].a += a
			cells[idx].n++
		}
	}

	nearest := func(r, g, b uint32) lipgloss.Color {
		return lipgloss.Color(uint8ToHex(uint8(r>>8), uint8(g>>8), uint8(b>>8)))
	}

	var buf strings.Builder
	for x := range gridW {
		top := cells[x]
		bottom := cells[gridW+x]
		if top.n == 0 || bottom.n == 0 || top.a/top.n < 128 || bottom.a/bottom.n < 128 {
			buf.WriteString(" ")
			continue
		}
		fg := nearest(top.r/top.n, top.g/top.n, top.b/top.n)
		bg := nearest(bottom.r/bottom.n, bottom.g/bottom.n, bottom.b/bottom.n)
		if top.a/top.n < 128 {
			fg = bg
			buf.WriteString(lipgloss.NewStyle().Background(bg).Render(" "))
		} else {
			buf.WriteString(lipgloss.NewStyle().Foreground(fg).Background(bg).Render("▀"))
		}
	}
	return buf.String()
}

func isPUA(s string) bool {
	for _, r := range s {
		if 0xE000 <= r && r <= 0xF8FF {
			return true
		}
	}
	return false
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
		entry, ok := emojiBank[shortcode]
		if ok && entry.glyph != "" && !isPUA(entry.glyph) {
			buf.WriteString(entry.glyph)
		} else if ok && entry.assetGlyph != "" {
			buf.WriteString(entry.assetGlyph)
		} else if ok {
			buf.WriteString(shortcode[1 : len(shortcode)-1])
		} else {
			buf.WriteString(shortcode)
		}
		i = pos + end + 2
	}
	return buf.String()
}
