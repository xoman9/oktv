package core

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

func TestHongguoPageProgressUsesNativeCursor(t *testing.T) {
	engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
		payload := nativeHongguoRequestForTest(t, r)
		base := map[string]int{"default": 1000, "comic_series": 2000, "ai_series": 3000}[payload.Scene]
		return nativeHongguoResponse(r, payload.Offset+18, true, "fixture", strconv.Itoa(base+payload.Offset)), nil
	})
	engine.downloader.cfg.MaxPagesPerSort = 2
	ctx := context.WithValue(context.Background(), backgroundCatalogKey{}, true)
	for index, input := range []nativeInput{{Source: sourceHongguo, Page: 1, Force: true}, {Source: sourceHongguo, Page: 3, Force: true}} {
		got, err := engine.nativeCatalog(ctx, input)
		want := (index + 1) * 2
		if err != nil || got.Page != want {
			t.Fatalf("page=%d want=%d err=%v", got.Page, want, err)
		}
		if status := engine.sourceStatus(sourceHongguo); status.Page != want {
			t.Fatalf("status page=%d want=%d", status.Page, want)
		}
	}
	engine.mu.Lock()
	state := engine.catalogStates[sourceHongguo]
	state.Page = 1
	engine.catalogStates[sourceHongguo] = state
	err := engine.writeCatalogDiskLocked()
	engine.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	restored := reopenCatalogEngine(t, engine.directory, nil)
	if status := restored.sourceStatus(sourceHongguo); status.Page != 4 {
		t.Fatalf("legacy page not restored: %+v", status)
	}
	exported := restored.nativeLibraryExport()
	target := exchangeFixtureEngine(t)
	if _, err := target.nativeLibraryImport(exported); err != nil {
		t.Fatal(err)
	}
	if status := target.sourceStatus(sourceHongguo); status.Page != 4 {
		t.Fatalf("import page not retained: %+v", status)
	}
}

func TestHongguoPageProgressKeepsPartialProgress(t *testing.T) {
	engine := exchangeFixtureEngine(t)
	cursor := engine.downloader.hongguoCatalogSnapshot()
	cursor.Feeds["short_play"] = hongguoCatalogCursor{Offset: 90, Initialized: true}
	result := nativeCatalogResult{Items: []nativeDrama{{ID: "hongguo:101", Source: sourceHongguo, Title: "合成影片"}}, Page: hongguoCatalogLoadedPage(cursor, ""), HasMore: true, Warning: "fixture timeout", hongguo: cursor}
	if err := engine.saveCatalogCache(sourceHongguo, &result); err != nil {
		t.Fatal(err)
	}
	if got := engine.sourceStatus(sourceHongguo); got.Page != 5 || !got.HasMore {
		t.Fatalf("partial page lost: %+v", got)
	}
	cursor.Feeds["category:ai_series"] = hongguoCatalogCursor{Offset: 126, Initialized: true}
	if got := hongguoCatalogLoadedPage(cursor, "ai_series"); got != 7 {
		t.Fatalf("category page=%d", got)
	}
	if got := hongguoCatalogLoadedPage(cursor, ""); got != 5 {
		t.Fatalf("category page leaked into base: %d", got)
	}
}
