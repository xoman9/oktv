package core

import (
	"context"
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestLiuguoCatalogRefreshUsesDonorCategoryAndPersists(t *testing.T) {
	var paths []string
	engine := sourceFixtureEngine(t, func(request *http.Request) (*http.Response, error) {
		paths = append(paths, request.URL.Path)
		switch request.URL.Path {
		case "/vodtype/3.html":
			return sourceFixtureResponse(request, 200, `<div class="video-item"><a href="/voddetail/101.html" title="合成影片甲">合成影片甲</a></div>`), nil
		case "/vodtype/3-2.html":
			return sourceFixtureResponse(request, 200, `<div class="media-card"><a href="/voddetail/102.html" title="合成影片乙">合成影片乙</a></div>`), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s", request.URL.Path)
		}
	})
	for _, page := range []int{1, 2} {
		if err := engine.loadSourceCatalogPage(context.Background(), sourceLiuguo, page); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(paths, []string{"/vodtype/3.html", "/vodtype/3-2.html"}) {
		t.Fatal(paths)
	}
	cached := engine.nativeCached(sourceLiuguo)
	if len(cached.Items) != 2 || cached.Page != 2 || !cached.HasMore {
		t.Fatalf("catalog not saved: %+v", cached)
	}
	restored := &nativeEngine{directory: engine.directory, catalogs: map[string][]nativeDrama{}, catalogStates: map[string]nativeCatalogState{}}
	restored.loadCatalogCache()
	if len(restored.nativeCached(sourceLiuguo).Items) != 2 {
		t.Fatal("catalog did not survive restart")
	}
}

func TestLiuguoCatalogExplicitCategoryAndEmptyPage(t *testing.T) {
	var paths []string
	engine := sourceFixtureEngine(t, func(request *http.Request) (*http.Response, error) {
		paths = append(paths, request.URL.Path)
		return sourceFixtureResponse(request, 200, `<html><h1>Site Unavailable</h1></html>`), nil
	})
	items, more, err := engine.downloader.fetchMaccmsCatalogPage(context.Background(), sourceLiuguo, 1, "23")
	if err == nil || len(items) != 0 || more {
		t.Fatal("unavailable page accepted as catalog")
	}
	if !reflect.DeepEqual(paths, []string{"/vodtype/23.html"}) {
		t.Fatalf("unexpected fallback requests: %v", paths)
	}
}

func TestLiuguoCatalogDonorNavigationAndAttributes(t *testing.T) {
	document, err := html.Parse(strings.NewReader(`<div class="video-item"><a href="/vodtype/3.html">导航</a><a href="/vodplay/901-1-1.html" title="&lt;b&gt;合成&lt;/b&gt;  标题">播放</a></div><div class="video-item"><a href="/vodplay/901-1-1.html">重复</a></div>`))
	if err != nil {
		t.Fatal(err)
	}
	items := liuguoCatalogCards(document, "https://example.test")
	if len(items) != 1 || items[0].SourceID != "901" || items[0].Title != "合成 标题" {
		t.Fatalf("donor parser mismatch: %+v", items)
	}
	document, _ = html.Parse(strings.NewReader(`<section><div><a href="/duanju/902.html" title="合成影片乙">详情</a></div></section>`))
	items = liuguoCatalogCards(document, "https://example.test")
	if len(items) != 1 || items[0].SourceID != "902" {
		t.Fatalf("donor anchor fallback mismatch: %+v", items)
	}
}

func TestLiuguoCatalogDonorHeadersAndTerminalPage(t *testing.T) {
	engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/vodtype/3-4.html" || r.Header.Get("Accept") != liuguoCatalogAccept || r.Header.Get("User-Agent") != duanjuUserAgent {
			t.Fatalf("donor request mismatch: %s %v", r.URL, r.Header)
		}
		return sourceFixtureResponse(r, 200, `<html><body></body></html>`), nil
	})
	items, more, err := engine.downloader.fetchLiuguoCatalogPage(context.Background(), 4, "")
	if err != nil || more || len(items) != 0 {
		t.Fatal("donor terminal-page behavior mismatch", err)
	}
}
