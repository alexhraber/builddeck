package tui

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"

	"github.com/alexhraber/builddeck/internal/buildkite"
)

//go:embed emoji-unicode.json
var emojiUnicodeJSON []byte

type emojiEntry struct {
	glyph string
}

var (
	emojiBank map[string]emojiEntry
	emojiMu   sync.RWMutex
)

var nerdFontIcons = map[string]string{
	"docker":                "\uf21b",                             //  fa-docker
	"go":                    "\ue627",                             //  dev-go
	"golang":                "\ue627",                             //  dev-go (GO letters)
	"python":                "\ue73c",                             //  dev-python
	"rust":                  "\ue7a8",                             // 
	"ruby":                  "\ue791",                             // 
	"node":                  "\ue718",                             //  dev-nodejs_small
	"npm":                   "\ue71e",                             //  dev-npm
	"github":                "\uf09b",                             //  fa-github
	"git":                   "\uf1d3",                             //  fa-git-alt
	"aws":                   "\uf0c2",                             //  fa-cloud
	"gcp":                   "\uf0c2",                             //  fa-cloud
	"azure":                 "\uf0c2",                             //  fa-cloud
	"linux":                 "\uf17c",                             //  fa-linux
	"apple":                 "\uf179",                             //  fa-apple
	"windows":               "\uf17a",                             //  fa-windows
	"test":                  "\uf478",                             //  oct-check
	"check":                 "\uf00c",                             //  fa-check
	"star":                  "\uf005",                             //  fa-star
	"heart":                 "\uf004",                             //  fa-heart
	"bug":                   "\uf188",                             //  fa-bug
	"rocket":                "\uf135",                             //  fa-rocket
	"shipit":                "\uf135",                             //  fa-rocket
	"warning":               "\uf071",                             //  fa-warning
	"lock":                  "\uf023",                             //  fa-lock
	"fire":                  "\uf06d",                             //  fa-fire
	"tada":                  "\uf0e7",                             //  fa-bolt (zap)
	"package":               "\uf187",                             //  fa-archive
	"art":                   "\uf1fc",                             //  fa-paint-brush
	"book":                  "\uf02d",                             //  fa-book
	"hammer":                "\uf0e7",                             //  fa-bolt
	"fast_forward":          "\u23e9",                             // ⏩
	"buildkite":             "\uf135",                             //  fa-rocket
	"sparkles":              "\uf00a",                             //  md-sparkles
	"merge":                 "\uf157",                             //  fa-code-fork
	"fork":                  "\uf126",                             //  fa-code-fork
	"terminal":              "\uf120",                             //  fa-terminal
	"zap":                   "\uf0e7",                             //  fa-bolt
	"lightning":             "\uf0e7",                             //  fa-bolt
	"gear":                  "\uf013",                             //  fa-gear
	"cog":                   "\uf013",                             //  fa-cog
	"wrench":                "\uf0ad",                             //  fa-wrench
	"tools":                 "\uf0ad",                             //  fa-wrench
	"building":              "\uf1ad",                             //  fa-building
	"office":                "\uf1ad",                             //  fa-building
	"home":                  "\uf015",                             //  fa-home
	"seedling":              "\uf4d8",                             //  oct-seedling
	"play":                  "\uf04b",                             //  fa-play
	"stop":                  "\uf04d",                             //  fa-stop
	"pause":                 "\uf04c",                             //  fa-pause
	"sync":                  "\uf021",                             //  fa-sync
	"refresh":               "\uf021",                             //  fa-sync
	"search":                "\uf002",                             //  fa-search
	"plus":                  "\uf067",                             //  fa-plus
	"pipeline":              "\uf0c5",                             //  fa-files-o
	"tag":                   "\uf02b",                             //  fa-tag
	"branch":                "\uf126",                             //  fa-code-fork
	"commit":                "\uf01c",                             //  fa-asterisk
	"pr":                    "\uf09b",                             //  md-source-pull-request
	"pull_request":          "\uf09b",                             //  md-source-pull-request
	"docker_compose":        "\uf21b",                             //  fa-docker
	"database":              "\uf1c0",                             //  fa-database
	"redis":                 "\ue76d",                             //  dev-redis
	"postgres":              "\ue76e",                             //  dev-postgres
	"postgresql":            "\ue76e",                             //  dev-postgres
	"mysql":                 "\ue704",                             //  dev-mysql
	"mongo":                 "\ue7a4",                             //  dev-mongodb
	"mongodb":               "\ue7a4",                             //  dev-mongodb
	"nginx":                 "\ue776",                             //  dev-nginx
	"nodejs":                "\ue718",                             //  dev-nodejs_small
	"typescript":            "\ue628",                             //  dev-typescript
	"ts":                    "\ue628",                             //  dev-typescript
	"javascript":            "\ue781",                             //  dev-javascript
	"js":                    "\ue781",                             //  dev-javascript
	"java":                  "\ue738",                             //  dev-java
	"kotlin":                "\ue789",                             //  dev-kotlin
	"swift":                 "\ue755",                             //  dev-swift
	"elixir":                "\ue62d",                             //  dev-elixir
	"cplusplus":             "\ue61d",                             //  dev-cpp
	"c":                     "\ue61d",                             //  dev-cpp
	"cpp":                   "\ue61d",                             //  dev-cpp
	"csharp":                "\ue648",                             //  dev-csharp
	"dotnet":                "\ue77f",                             //  dev-dotnet
	"html":                  "\ue736",                             //  dev-html5
	"css":                   "\ue749",                             //  dev-css3
	"json":                  "\ue60b",                             //  dev-json
	"yaml":                  "\ue60b",                             //  dev-yaml
	"markdown":              "\uf48a",                             //  oct-markdown
	"php":                   "\ue73d",                             //  dev-php
	"scala":                 "\ue737",                             //  dev-scala
	"lua":                   "\ue620",                             //  dev-lua
	"haskell":               "\ue61f",                             //  dev-haskell
	"clojure":               "\ue76a",                             //  dev-clojure
	"erlang":                "\ue7b1",                             //  dev-erlang
	"dart":                  "\ue798",                             //  dev-dart
	"flutter":               "\ue798",                             //  dev-dart
	"deno":                  "\ue60f",                             //  dev-deno
	"bun":                   "\ue76b",                             //  dev-bun
	"k8s":                   "\uf10b",                             //  md-kubernetes
	"terraform":             "\u2442",                             // ⑂ (actually \ue60b dev-terraform)
	"tf":                    "\ue60b",                             //  dev-terraform
	"ansible":               "\uf19b",                             //  fa-ansible (FA only, actual codepoint may vary)
	"circleci":              "\ue702",                             //  dev-circleci
	"travis":                "\ue77e",                             //  dev-travis
	"bitbucket":             "\ue703",                             //  dev-bitbucket
	"gitlab":                "\uf296",                             //  fa-gitlab
	"jira":                  "\uf17d",                             //  fa-jira
	"email":                 "\uf0e0",                             //  fa-envelope
	"mail":                  "\uf0e0",                             //  fa-envelope
	"chat":                  "\uf075",                             //  fa-comment
	"phone":                 "\uf095",                             //  fa-phone
	"camera":                "\uf030",                             //  fa-camera
	"video":                 "\uf03d",                             //  fa-video-camera
	"youtube":               "\uf167",                             //  fa-youtube
	"twitter":               "\uf099",                             //  fa-twitter
	"linkedin":              "\uf0e1",                             //  fa-linkedin
	"slack":                 "\uf198",                             //  fa-slack
	"clock":                 "\uf017",                             //  fa-clock-o
	"time":                  "\uf017",                             //  fa-clock-o
	"calendar":              "\uf073",                             //  fa-calendar
	"date":                  "\uf073",                             //  fa-calendar
	"eye":                   "\uf06e",                             //  fa-eye
	"graph":                 "\uf080",                             //  fa-bar-chart
	"chart":                 "\uf080",                             //  fa-bar-chart
	"download":              "\uf019",                             //  fa-download
	"upload":                "\uf093",                             //  fa-upload
	"link":                  "\uf0c1",                             //  fa-link
	"url":                   "\uf0c1",                             //  fa-link
	"globe":                 "\uf0ac",                             //  fa-globe
	"world":                 "\uf0ac",                             //  fa-globe
	"flag":                  "\uf024",                             //  fa-flag
	"trophy":                "\uf091",                             //  fa-trophy
	"medal":                 "\uf0a3",                             //  fa-medal
	"award":                 "\uf0a3",                             //  fa-medal
	"thumbsup":              "\uf164",                             //  fa-thumbs-up
	"thumbsdown":            "\uf165",                             //  fa-thumbs-down
	"handshake":             "\uf2b5",                             //  fa-handshake-o
	"users":                 "\uf0c0",                             //  fa-users
	"person":                "\uf007",                             //  fa-user
	"document":              "\uf016",                             //  fa-file-o
	"file":                  "\uf016",                             //  fa-file-o
	"folder":                "\uf07b",                             //  fa-folder
	"archive":               "\uf187",                             //  fa-archive
	"zip":                   "\uf187",                             //  fa-archive
	"key":                   "\uf084",                             //  fa-key
	"secret":                "\uf084",                             //  fa-key
	"shirt":                 "\uf553",                             //  fa-tshirt
	"ladybug":               "\uf188",                             //  fa-bug
	"beetle":                "\uf188",                             //  fa-bug
	"ant":                   "\uf188",                             //  fa-bug
	"snail":                 "\uf188",                             //  fa-bug
	"turtle":                "\uf188",                             //  fa-bug
	"dog":                   "\uf6d3",                             //  fa-dog
	"cat":                   "\uf6be",                             //  fa-cat
	"whale":                 "\uf72c",                             //  fa-whale
	"unicorn":               "\uf15b",                             //  fa-html5 (no unicorn in FA)
	"robot":                 "\uf544",                             //  fa-robot
	"alien":                 "\uf8df",                             //  fa-alien
	"broom":                 "\U0001F9F9",                         // 🧹
	"detective":             "\U0001F575\uFE0F",                   // 🕵️
	"male_detective":        "\U0001F575\uFE0F\u200D\u2642\uFE0F", // 🕵️‍♂️
	"male-detective":        "\U0001F575\uFE0F\u200D\u2642\uFE0F", // 🕵️‍♂️
	"female_detective":      "\U0001F575\uFE0F\u200D\u2640\uFE0F", // 🕵️‍♀️
	"female-detective":      "\U0001F575\uFE0F\u200D\u2640\uFE0F", // 🕵️‍♀️
	"building_construction": "\U0001F3D7",                         // 🏗
	"building-construction": "\U0001F3D7",                         // 🏗
	"construction":          "\U0001F6A7",                         // 🚧
	"test_tube":             "\U0001F9EA",                         // 🧪
	"test-tube":             "\U0001F9EA",                         // 🧪
	"white_check_mark":      "\u2705",                             // ✅
	"white-check-mark":      "\u2705",                             // ✅
	"x":                     "\u274C",                             // ❌
	"cross_mark":            "\u274C",                             // ❌
	"green_heart":           "\U0001F49A",                         // 💚
	"green-heart":           "\U0001F49A",                         // 💚
	"boom":                  "\U0001F4A5",                         // 💥
	"collision":             "\U0001F4A5",                         // 💥
	"recycle":               "\u267B\uFE0F",                       // ♻️
	"pencil":                "\u270F\uFE0F",                       // ✏️
	"memo":                  "\U0001F4DD",                         // 📝
	"books":                 "\U0001F4DA",                         // 📚
	"arrow_up":              "\u2B06",                             // ⬆
	"arrow-up":              "\u2B06",                             // ⬆
	"arrow_down":            "\u2B07",                             // ⬇
	"arrow-down":            "\u2B07",                             // ⬇
	"arrow_up_down":         "\u2195",                             // ↕
	"heavy_plus_sign":       "\u2795",                             // ➕
	"heavy-plus-sign":       "\u2795",                             // ➕
	"heavy_minus_sign":      "\u2796",                             // ➖
	"heavy-minus-sign":      "\u2796",                             // ➖
	"heavy_check_mark":      "\u2714",                             // ✔
	"heavy-check-mark":      "\u2714",                             // ✔
	"wave":                  "\U0001F44B",                         // 👋
	"beers":                 "\U0001F37B",                         // 🍻
	"sweat_smile":           "\U0001F605",                         // 😅
	"sweat-smile":           "\U0001F605",                         // 😅
	"smile":                 "\U0001F604",                         // 😄
	"sob":                   "\U0001F62D",                         // 😭
	"scream":                "\U0001F631",                         // 😱
	"buildkite_party":       "\U0001F973",                         // 🥳
	"partying_face":         "\U0001F973",                         // 🥳
	"partying-face":         "\U0001F973",                         // 🥳
	"facepalm":              "\U0001F926",                         // 🤦
}

func init() {
	emojiBank = make(map[string]emojiEntry, len(nerdFontIcons)+200)
	for name, glyph := range nerdFontIcons {
		emojiBank[":"+name+":"] = emojiEntry{glyph: glyph}
	}
	loadUnicodeEmojiMap()
}

func loadUnicodeEmojiMap() {
	var entries map[string]string
	if err := json.Unmarshal(emojiUnicodeJSON, &entries); err != nil {
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
	defer emojiMu.Unlock()
	for _, e := range apiEmojis {
		key := ":" + e.Name + ":"
		if _, exists := emojiBank[key]; !exists {
			emojiBank[key] = emojiEntry{}
		}
	}
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
		if ok && entry.glyph != "" {
			buf.WriteString(entry.glyph)
			buf.WriteByte('\t')
		} else if ok {
			buf.WriteString(shortcode[1 : len(shortcode)-1])
		} else {
			buf.WriteString(shortcode)
		}
		i = pos + end + 2
	}
	return buf.String()
}
