package core

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
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
	items, more, err := engine.downloader.fetchMaccmsCatalogPage(context.Background(), sourceLiuguo, 2, "23")
	if err == nil || len(items) != 0 || more {
		t.Fatal("unavailable page accepted as catalog")
	}
	if !reflect.DeepEqual(paths, []string{"/vodtype/23-2.html"}) {
		t.Fatalf("unexpected fallback requests: %v", paths)
	}
}
