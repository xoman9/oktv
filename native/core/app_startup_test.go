package core

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

func TestHongguoStartupBoundsForegroundWithoutLimitingUpdates(t *testing.T) {
	for _, background := range []bool{false, true} {
		calls := 0
		engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
			calls++
			payload := nativeHongguoRequestForTest(t, r)
			return nativeHongguoResponse(r, payload.Offset+18, true, "fixture", strconv.Itoa(payload.Offset+101), strconv.Itoa(payload.Offset+102)), nil
		})
		engine.downloader.cfg.MaxPagesPerSort = 2
		ctx := context.Background()
		if background {
			ctx = context.WithValue(ctx, backgroundCatalogKey{}, true)
		}
		_, err := engine.nativeCatalog(ctx, nativeInput{Source: sourceHongguo, Force: true})
		if err != nil {
			t.Fatal(err)
		}
		want := 3
		if background {
			want = 6
		}
		if calls != want {
			t.Fatalf("background=%v calls=%d want=%d", background, calls, want)
		}
	}
}
