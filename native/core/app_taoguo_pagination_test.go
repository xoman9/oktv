package core

import (
	"context"
	"net/http"
	"testing"
)

func TestTaoguoPaginationStopsWithoutNewItems(t *testing.T) {
	for _, tc := range []struct {
		name, body  string
		more        bool
		count, page int
	}{
		{"empty", `<html></html>`, false, 1, 7},
		{"duplicate", `<div class="video-item"><a href="/voddetail/101.html" title="合成影片甲">合成影片甲</a></div>`, false, 1, 7},
		{"new", `<div class="video-item"><a href="/voddetail/102.html" title="合成影片乙">合成影片乙</a></div>`, true, 2, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
				calls++
				return sourceFixtureResponse(r, 200, tc.body), nil
			})
			if err := engine.saveCatalogCache(sourceTaoguo, &nativeCatalogResult{Items: []nativeDrama{{ID: "taoguo:101", Source: sourceTaoguo, Title: "合成影片甲"}}, Page: 7, HasMore: true}); err != nil {
				t.Fatal(err)
			}
			if err := engine.loadMoreSourceCatalogPages(context.Background(), sourceTaoguo, 1); err != nil {
				t.Fatal(err)
			}
			status := engine.sourceStatus(sourceTaoguo)
			if status.Page != tc.page || status.Count != tc.count || status.HasMore != tc.more {
				t.Fatalf("status=%+v", status)
			}
			if !tc.more {
				before := calls
				if err := engine.loadMoreSourceCatalogPages(context.Background(), sourceTaoguo, 10); err != nil {
					t.Fatal(err)
				}
				if calls != before {
					t.Fatal("continued requesting exhausted pages")
				}
				restored := reopenCatalogEngine(t, engine.directory, nil)
				if got := restored.sourceStatus(sourceTaoguo); got.Page != 7 || got.Count != 1 || got.HasMore {
					t.Fatalf("restart lost terminal state: %+v", got)
				}
			}
		})
	}
}

func TestTaoguoPaginationFailureDoesNotMarkExhausted(t *testing.T) {
	engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
		return sourceFixtureResponse(r, http.StatusServiceUnavailable, "fixture failure"), nil
	})
	if err := engine.saveCatalogCache(sourceTaoguo, &nativeCatalogResult{Items: []nativeDrama{{ID: "taoguo:101", Source: sourceTaoguo, Title: "合成影片甲"}}, Page: 7, HasMore: true}); err != nil {
		t.Fatal(err)
	}
	if err := engine.loadMoreSourceCatalogPages(context.Background(), sourceTaoguo, 1); err == nil {
		t.Fatal("network failure was hidden")
	}
	if status := engine.sourceStatus(sourceTaoguo); status.Page != 7 || status.Count != 1 || !status.HasMore {
		t.Fatalf("failure marked catalog exhausted: %+v", status)
	}
}
