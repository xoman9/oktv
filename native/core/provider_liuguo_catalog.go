package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type liuguoCatalogRequestKey struct{}

const liuguoCatalogAccept = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"

var (
	liuguoCatalogNavigation = regexp.MustCompile(`(?i)/(?:[a-z]*type|search|vodshow|label|gbook|rss|map|topic|actor|show/by)/|\?class=`)
	liuguoCatalogDetail     = regexp.MustCompile(`(?i)/(?:index\.php/vod/)?(?:voddetail|detail|vod|show|view|movie|drama|duanju|zywview|xzyxvd)/`)
	liuguoCatalogID         = regexp.MustCompile(`(?i)/(?:voddetail|vodplay|djplay|detail|show|vod|drama|duanju|movie|tv|id)/([0-9]+)(?:[-./]|$)`)
	liuguoCatalogMarkup     = regexp.MustCompile(`<[^>]*>`)
)

var liuguoCatalogClasses = []string{
	"col-lg-2", "col-xl-2", "module-item", "module-poster-item", "public-list-box", "videoBox",
	"detail-list-item", "col-md-6", "col-6", "listItem", "FeaturedList_featuredItem", "BrowseList_listItem",
	"SecondList_secondListItem", "vodlist__item", "v_list", "entry-wrapper", "TagBookList_tagItem",
	"video-item", "vodlist_item", "myui-vodlist__box", "thumbnail", "media-card",
}

func (d *Downloader) fetchLiuguoCatalogPage(ctx context.Context, page int, category string) ([]Drama, bool, error) {
	base := d.duanjuBaseURL(sourceLiuguo)
	class := strings.TrimSpace(category)
	if class == "" || class == "all" {
		class = "3"
	}
	if !webProviderNumericID.MatchString(class) {
		return nil, false, errors.New("榴果分类编号无效")
	}
	address := fmt.Sprintf("%s/vodtype/%s.html", base, url.PathEscape(class))
	if page > 1 {
		address = fmt.Sprintf("%s/vodtype/%s-%d.html", base, url.PathEscape(class), page)
	}
	ctx = context.WithValue(ctx, liuguoCatalogRequestKey{}, true)
	document, _, err := d.fetchProviderPage(ctx, address, base+"/", duanjuUserAgent)
	if err != nil {
		return nil, false, err
	}
	items := liuguoCatalogCards(document, base)
	if len(items) == 0 && page <= 1 {
		anchors := providerHTMLNodes(document, func(n *html.Node) bool { return n.Data == "a" })
		candidates := 0
		for _, anchor := range anchors {
			if liuguoCatalogDetail.MatchString(providerHTMLAttr(anchor, "href")) {
				candidates++
			}
		}
		return nil, false, fmt.Errorf("榴果原生目录未解析到影片（分类 %s、链接 %d、详情链接 %d）", class, len(anchors), candidates)
	}
	return items, len(items) > 0, nil
}

func liuguoCatalogCards(document *html.Node, base string) []Drama {
	var cards []*html.Node
	seen := map[*html.Node]bool{}
	for _, name := range liuguoCatalogClasses {
		for _, node := range providerHTMLNodes(document, maccmsClassMatcher(name)) {
			if !seen[node] {
				seen[node] = true
				cards = append(cards, node)
			}
		}
	}
	if len(cards) == 0 {
		seen = map[*html.Node]bool{}
		for _, anchor := range providerHTMLNodes(document, func(n *html.Node) bool {
			return n.Data == "a" && liuguoCatalogDetail.MatchString(providerHTMLAttr(n, "href"))
		}) {
			for parent := anchor.Parent; parent != nil && parent.Type == html.ElementNode; parent = parent.Parent {
				if seen[parent] {
					break
				}
				if parent.Data != "li" && parent.Data != "div" {
					continue
				}
				seen[parent] = true
				cards = append(cards, parent)
			}
		}
	}
	var items []Drama
	ids := map[string]bool{}
	for _, card := range cards {
		link := liuguoCatalogCardLink(card, base)
		title := liuguoCatalogCardTitle(card)
		if link == "" || title == "" {
			continue
		}
		id := liuguoCatalogSourceID(link)
		if id == "" || ids[id] {
			continue
		}
		ids[id] = true
		remark := maccmsCardRemark(card)
		items = append(items, Drama{ID: providerDramaID(sourceLiuguo, id), Source: sourceLiuguo, SourceID: id, Title: title,
			Cover: liuguoCatalogCardCover(card, link), Remark: remark, EpisodeCount: json.Number(strconv.Itoa(duanjuEpisodeNumber(remark, 0))), ChannelName: duanjuSourceName(sourceLiuguo)})
	}
	return items
}

func liuguoCatalogCardLink(card *html.Node, base string) string {
	for _, anchor := range providerHTMLNodes(card, func(n *html.Node) bool { return n.Data == "a" }) {
		link := providerHTMLAttr(anchor, "href")
		if link == "" || strings.Contains(link, "javascript:") || liuguoCatalogNavigation.MatchString(link) {
			continue
		}
		if address := duanjuAbsolute(base, link); address != "" {
			return address
		}
	}
	return ""
}

func liuguoCatalogCardTitle(card *html.Node) string {
	for _, attribute := range []string{"title", "alt"} {
		for _, node := range providerHTMLNodes(card, func(n *html.Node) bool { return n.Data == "a" || n.Data == "img" }) {
			value := liuguoCatalogMarkup.ReplaceAllString(providerHTMLAttr(node, attribute), "")
			value = html.UnescapeString(value)
			if value = strings.Join(strings.Fields(value), " "); value != "" {
				return value
			}
		}
	}
	for _, node := range providerHTMLNodes(card, func(n *html.Node) bool { return n.Data == "a" }) {
		if value := providerHTMLText(node); value != "" {
			return value
		}
	}
	return ""
}

func liuguoCatalogCardCover(card *html.Node, pageURL string) string {
	for _, tag := range []string{"img", "a"} {
		for _, node := range providerHTMLNodes(card, func(n *html.Node) bool { return n.Data == tag }) {
			attributes := []string{"data-original", "data-src", "src"}
			if tag == "a" {
				attributes = attributes[:2]
			}
			for _, attribute := range attributes {
				if address := providerCoverAddress(providerHTMLAttr(node, attribute), pageURL); address != "" {
					return address
				}
			}
		}
	}
	return ""
}

func liuguoCatalogSourceID(link string) string {
	parsed, err := url.Parse(link)
	if err != nil {
		return ""
	}
	path := strings.TrimSuffix(parsed.Path, ".html")
	if matches := liuguoCatalogID.FindStringSubmatch(path + "/"); len(matches) > 1 {
		return matches[1]
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for index := len(parts) - 1; index >= 0; index-- {
		candidate := strings.TrimSuffix(parts[index], ".html")
		if webProviderNumericID.MatchString(candidate) {
			return candidate
		}
		if _, tail, found := strings.Cut(candidate, "-"); found && webProviderNumericID.MatchString(tail) {
			return tail
		}
	}
	if id := parsed.Query().Get("id"); webProviderNumericID.MatchString(id) {
		return id
	}
	return ""
}
