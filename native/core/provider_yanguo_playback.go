package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var yanguo171IDPattern = regexp.MustCompile(`(?i)/(?:voddetail|vodplay|detail|show|vod|drama|movie|tv|id)/([0-9]+)(?:[-./]|$)`)
var yanguo171PlayerPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\$\.url\s*=\s*"([^"]+)"`),
	regexp.MustCompile(`(?i)const\s+source\s*=\s*'([^']+)'`),
	regexp.MustCompile(`(?i)"url"\s*:\s*"([^"]+\.(?:m3u8|mp4)[^"]*)"`),
	regexp.MustCompile(`(?i)(https?://[^\s"'<>]+\.(?:m3u8|mp4)[^\s"'<>]*)`),
}

func yanguo171SourceID(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	path := strings.TrimSuffix(parsed.Path, ".html")
	if match := yanguo171IDPattern.FindStringSubmatch(path + "/"); len(match) > 1 {
		return match[1]
	}
	if match := maccmsRouteMyui.FindStringSubmatch(parsed.Path); len(match) > 3 {
		return match[1]
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for index := len(parts) - 1; index >= 0; index-- {
		candidate := parts[index]
		if webProviderNumericID.MatchString(candidate) {
			return candidate
		}
		if _, tail, ok := strings.Cut(candidate, "-"); ok && webProviderNumericID.MatchString(tail) {
			return tail
		}
	}
	if id := parsed.Query().Get("id"); webProviderNumericID.MatchString(id) {
		return id
	}
	return ""
}

func yanguo171Episodes(episodes []providerEpisode, sourceID string) []providerEpisode {
	var strict, matched []providerEpisode
	identified := false
	for _, episode := range episodes {
		id := yanguo171SourceID(episode.URL)
		if id == "" {
			continue
		}
		identified = true
		if id != sourceID {
			continue
		}
		matched = append(matched, episode)
		if maccmsEpisodeLink.MatchString(episode.URL) {
			strict = append(strict, episode)
		}
	}
	if len(strict) > 0 {
		return strict
	}
	if len(matched) > 0 {
		return matched
	}
	if identified {
		return nil
	}
	return episodes
}

func yanguo171Unavailable(document *html.Node) bool {
	for _, title := range providerHTMLNodes(document, func(node *html.Node) bool { return node.Data == "title" }) {
		if strings.EqualFold(strings.TrimSpace(providerHTMLText(title)), "Site Unavailable") {
			return true
		}
	}
	return false
}

func yanguo171SinglePageEpisode(document *html.Node, pageURL, sourceID string) []providerEpisode {
	if !maccmsEpisodeLink.MatchString(pageURL) || yanguo171SourceID(pageURL) != sourceID {
		return nil
	}
	canonicalID := ""
	for _, node := range providerHTMLNodes(document, func(node *html.Node) bool {
		return node.Data == "link" && strings.EqualFold(providerHTMLAttr(node, "rel"), "canonical") ||
			node.Data == "meta" && strings.EqualFold(providerHTMLAttr(node, "property"), "og:url")
	}) {
		raw := firstNonEmpty(providerHTMLAttr(node, "href"), providerHTMLAttr(node, "content"))
		if id := yanguo171SourceID(raw); id != "" {
			if id != sourceID {
				return nil
			}
			canonicalID = id
		}
	}
	for _, script := range providerHTMLNodes(document, func(node *html.Node) bool { return node.Data == "script" }) {
		var text strings.Builder
		for child := script.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.TextNode {
				text.WriteString(child.Data)
			}
		}
		if canonicalID == sourceID {
			for _, match := range yanguo171PlayerPatterns[1].FindAllStringSubmatch(text.String(), -1) {
				if len(match) < 2 {
					continue
				}
				address := maccmsNormalizePlaybackURL(match[1])
				parsed, err := url.Parse(address)
				if err == nil && isProviderHTTPMediaURL(address) && !strings.ContainsAny(address, "{}") && (strings.HasSuffix(strings.ToLower(parsed.Path), ".m3u8") || strings.HasSuffix(strings.ToLower(parsed.Path), ".mp4")) {
					return []providerEpisode{{Key: "1", Title: "正片", URL: pageURL, Index: 1}}
				}
			}
		}
		matches := maccmsPlayerData.FindStringSubmatch(text.String())
		if len(matches) < 2 {
			continue
		}
		var fields map[string]any
		if json.Unmarshal([]byte(matches[1]), &fields) != nil {
			continue
		}
		identity := mapString(fields, "vod_id", "vodId", "series_id", "id")
		if identity != "" && identity != sourceID {
			continue
		}
		linkID := yanguo171SourceID(mapString(fields, "link"))
		if linkID != "" && linkID != sourceID {
			continue
		}
		if identity != sourceID && linkID != sourceID && canonicalID != sourceID {
			continue
		}
		address := maccmsDecryptPlayerURL(mapString(fields, "url"), mapString(fields, "encrypt"))
		if !isProviderHTTPMediaURL(address) || strings.ContainsAny(address, "{}") {
			continue
		}
		return []providerEpisode{{Key: "1", Title: "正片", URL: pageURL, Index: 1}}
	}
	return nil
}

func yanguo171PlayerURL(body string) string {
	valid := func(raw string) string {
		candidate := maccmsNormalizePlaybackURL(strings.ReplaceAll(strings.TrimSpace(raw), `\/`, `/`))
		if strings.ContainsAny(candidate, "{}") || !isProviderHTTPMediaURL(candidate) {
			return ""
		}
		return candidate
	}
	fields := maccmsPlayerFields(body)
	if candidate := valid(maccmsDecryptPlayerURL(fields["url"], fields["encrypt"])); candidate != "" {
		return candidate
	}
	for _, pattern := range yanguo171PlayerPatterns {
		for _, match := range pattern.FindAllStringSubmatch(body, -1) {
			if len(match) > 1 {
				if candidate := valid(match[1]); candidate != "" {
					return candidate
				}
			}
		}
	}
	return ""
}

func (d *Downloader) resolveYanguo171Page(ctx context.Context, pageURL, referer string) (providerMedia, error) {
	responses := &playbackResponseURLs{}
	requestCtx := context.WithValue(ctx, playbackResponseURLsKey{}, responses)
	body, err := d.fetchProviderText(requestCtx, pageURL, referer)
	if err != nil {
		return providerMedia{}, err
	}
	if actual, ok := responses.values.Load(pageURL); ok {
		expected, found := yanguo171SourceID(pageURL), yanguo171SourceID(actual.(string))
		if expected != "" && found != "" && expected != found {
			return providerMedia{}, errors.New("艳果播放页跳转到了其他影片，请刷新剧集")
		}
	}
	current := yanguo171PlayerURL(body)
	for depth := 0; depth < 2 && current != ""; depth++ {
		parsed, err := url.Parse(current)
		if err != nil {
			break
		}
		path := strings.ToLower(parsed.Path)
		if strings.HasSuffix(path, ".m3u8") || strings.HasSuffix(path, ".mp4") || strings.HasSuffix(path, ".flv") {
			return providerMedia{URL: current, Referer: pageURL}, nil
		}
		body, err = d.fetchProviderText(ctx, current, pageURL)
		if err != nil {
			return providerMedia{}, err
		}
		next := yanguo171PlayerURL(body)
		if next == "" || next == current {
			break
		}
		current = next
	}
	return providerMedia{}, fmt.Errorf("艳果未返回可识别的视频地址，请刷新剧集后重试")
}
