package core

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestYanguo171CategoriesAndPageRoutes(t *testing.T) {
	var paths []string
	d, server := duanjuFixtureDownloader(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		fmt.Fprint(w, `<div class="module-item"><a href="/index.php/vod/play/id/123.html" title="合成影片">合成影片</a></div>`)
	})
	d.providerHosts[sourceYanguo] = server.URL
	categories, err := d.fetchDuanjuCategories(context.Background(), sourceYanguo)
	if err != nil || len(categories) != 21 {
		t.Fatalf("categories=%v err=%v", categories, err)
	}
	seen := map[string]bool{}
	for _, category := range categories {
		if seen[category.ID] || category.ID == "all" || category.ID == "1" {
			t.Fatalf("duplicate or placeholder category: %+v", category)
		}
		seen[category.ID] = true
	}
	for _, id := range []string{"44", "61", "72", "54"} {
		if !seen[id] || !validNativeCategory(sourceYanguo, id) {
			t.Fatalf("missing category %s", id)
		}
	}
	if len(paths) != 0 {
		t.Fatalf("static categories made network requests: %v", paths)
	}
	for _, page := range []int{1, 2} {
		items, _, err := d.fetchMaccmsCatalogPage(context.Background(), sourceYanguo, page, "61")
		if err != nil || len(items) != 1 || items[0].SourceID != "123" {
			t.Fatalf("page=%d items=%v err=%v", page, items, err)
		}
	}
	if len(paths) != 2 || paths[0] != "/index.php/vod/type/id/61.html" || paths[1] != "/index.php/vod/type/id/61/page/2.html" {
		t.Fatalf("unexpected requests: %v", paths)
	}
}
