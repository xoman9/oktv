package core

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"
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

func TestHongguoForegroundRequestBudget(t *testing.T) {
	engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 6*time.Second {
			t.Fatal("foreground request has no short deadline")
		}
		payload := nativeHongguoRequestForTest(t, r)
		return nativeHongguoResponse(r, payload.Offset+18, true, "fixture", "700001"), nil
	})
	_, err := engine.nativeCatalog(context.Background(), nativeInput{Source: sourceHongguo, Force: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHongguoForegroundEmptyBodyRetriesAreBounded(t *testing.T) {
	calls := 0
	engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
		calls++
		return sourceFixtureResponse(r, 200, ""), nil
	})
	engine.downloader.cfg.Retries = 5
	_, err := engine.downloader.hongguoAppRequest(context.Background(), http.MethodPost, "/reading/distribution/category/landpage/v/", nil, map[string]any{})
	if err == nil || calls != 2 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestHongguoForegroundCatalogLockCanBeCanceled(t *testing.T) {
	engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
		t.Error("request made while catalog locked")
		return nil, context.Canceled
	})
	client := engine.downloader.hongguoClient()
	client.catalogMu.Lock()
	defer client.catalogMu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := engine.downloader.fetchHongguoAppCatalog(ctx); done <- err }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled foreground still waiting for catalog lock")
	}
}

func TestHongguoForegroundPartialCatalogSurvivesDeadline(t *testing.T) {
	calls := 0
	engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nativeHongguoResponse(r, 18, true, "fixture", "700001"), nil
		}
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	result, err := engine.nativeCatalog(ctx, nativeInput{Source: sourceHongguo, Force: true})
	if err != nil || len(result.Items) != 1 || result.Warning == "" {
		t.Fatalf("partial results discarded: %+v err=%v", result, err)
	}
	if cached := engine.nativeCached(sourceHongguo); len(cached.Items) != 1 {
		t.Fatal("partial catalog not preserved")
	}
}
