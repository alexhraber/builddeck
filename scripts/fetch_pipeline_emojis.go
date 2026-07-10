//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run scripts/fetch_pipeline_emojis.go <assets-cookie>\n")
		fmt.Fprintf(os.Stderr, "  Cookie from browser: _buildkite_assets_session=<value>\n")
		os.Exit(1)
	}
	cookie := os.Args[1]
	dest := filepath.Join("assets", "pipeline-emojis")
	os.MkdirAll(dest, 0755)

	// Fetch the emoji JS bundle from the webapp
	jsURL := "https://buildkiteassets.com/frontend/emoji-v2-f87fbf85.js"
	fmt.Println("Fetching emoji JS bundle...")
	req, _ := http.NewRequest("GET", jsURL, nil)
	req.Header.Set("Cookie", "_buildkite_assets_session="+cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch bundle: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	jsData, _ := io.ReadAll(resp.Body)

	// Extract pipeline emoji names from img-buildkite-64 references
	re := regexp.MustCompile(`img-buildkite-64/([^"']+)\.png`)
	matches := re.FindAllStringSubmatch(string(jsData), -1)
	names := make(map[string]bool)
	for _, m := range matches {
		names[m[1]] = true
	}
	fmt.Printf("Found %d pipeline emoji names\n", len(names))

	// Download missing PNGs
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	var mu sync.Mutex
	downloaded := 0
	skipped := 0

	for name := range names {
		path := filepath.Join(dest, name+".png")
		if _, err := os.Stat(path); err == nil {
			skipped++
			continue
		}
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			url := fmt.Sprintf("https://buildkiteassets.com/emojis/img-buildkite-64/%s.png", n)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Cookie", "_buildkite_assets_session="+cookie)
			resp, err := http.DefaultClient.Do(req)
			if err != nil || resp.StatusCode != 200 {
				return
			}
			defer resp.Body.Close()
			data, _ := io.ReadAll(resp.Body)

			mu.Lock()
			downloaded++
			mu.Unlock()
			os.WriteFile(path, data, 0644)
			fmt.Printf("  ✓ %s (%d bytes)\n", n, len(data))
		}(name)
	}
	wg.Wait()

	fmt.Printf("\nDownloaded: %d, Skipped (exists): %d\n", downloaded, skipped)

	// Extract Unicode emoji mapping from the JSON.parse in the bundle
	jsonRe := regexp.MustCompile(`JSON\.parse\('(.+?)'\)`)
	jsonMatch := jsonRe.FindStringSubmatch(string(jsData))
	if len(jsonMatch) < 2 {
		fmt.Println("Could not find Unicode emoji data")
		return
	}
	raw := jsonMatch[1]
	raw = strings.NewReplacer("\\'", "'", `\"`, `"`, `\\`, `\`).Replace(raw)
	raw = decodeUnicodeEscapes(raw)

	var bundle struct {
		Emoji []struct {
			Name    string `json:"name"`
			Unicode string `json:"unicode,omitempty"`
			URL     string `json:"url,omitempty"`
		} `json:"emoji"`
	}
	if err := json.Unmarshal([]byte(raw), &bundle); err != nil {
		fmt.Fprintf(os.Stderr, "parse unicode: %v\n", err)
		return
	}

	unicodeMap := make(map[string]string)
	for _, e := range bundle.Emoji {
		if e.Unicode != "" {
			if _, exists := unicodeMap[e.Name]; !exists {
				unicodeMap[e.Name] = e.Unicode
			}
		}
	}
	fmt.Printf("Unicode emoji entries: %d\n", len(unicodeMap))

	out, _ := json.MarshalIndent(unicodeMap, "", "  ")
	os.WriteFile(filepath.Join(dest, "emoji-unicode.json"), out, 0644)
	fmt.Println("Written emoji-unicode.json")
}

func decodeUnicodeEscapes(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '\\' {
			switch s[i+1] {
			case 'u':
				if i+5 < len(s) {
					cp := parseHex(s[i+2 : i+6])
					if 0xD800 <= cp && cp <= 0xDBFF && i+11 < len(s) && s[i+6:i+8] == "\\u" {
						cp2 := parseHex(s[i+8 : i+12])
						full := 0x10000 + (cp-0xD800)*0x400 + (cp2 - 0xDC00)
						result.WriteRune(rune(full))
						i += 12
					} else {
						result.WriteRune(rune(cp))
						i += 6
					}
				} else {
					result.WriteByte(s[i])
					i++
				}
			case 'x':
				if i+3 < len(s) {
					result.WriteRune(rune(parseHex(s[i+2 : i+4])))
					i += 4
				} else {
					result.WriteByte(s[i])
					i++
				}
			default:
				result.WriteByte(s[i])
				i++
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

func parseHex(s string) int {
	var v int
	for _, c := range s {
		v *= 16
		switch {
		case '0' <= c && c <= '9':
			v += int(c - '0')
		case 'a' <= c && c <= 'f':
			v += int(c - 'a' + 10)
		case 'A' <= c && c <= 'F':
			v += int(c - 'A' + 10)
		}
	}
	return v
}
