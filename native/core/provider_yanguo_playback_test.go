package core

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestYanguo171SelectedMovieAndEpisode(t *testing.T) {
	var calls []string
	d, server := duanjuFixtureDownloader(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/index.php/vod/type/id/1.html":
			fmt.Fprint(w, `<div class="module-item"><a href="/index.php/vod/play/id/100/sid/1/nid/1.html" title="合成影片甲">合成影片甲</a></div><div class="module-item"><a href="/index.php/vod/play/id/200/sid/1/nid/1.html" title="合成影片乙">合成影片乙</a></div>`)
		case "/index.php/vod/play/id/100.html":
			fmt.Fprint(w, `<title>合成影片甲</title><div class="playlist"><a href="/index.php/vod/play/id/999/sid/1/nid/1.html">第1集</a><a href="/index.php/vod/play/id/100/sid/1/nid/1.html">第1集</a><a href="/index.php/vod/play/id/100/sid/1/nid/2.html">第2集</a><a href="/index.php/vod/play/id/200/sid/1/nid/1.html">第1集</a></div>`)
		case "/index.php/vod/play/id/100/sid/1/nid/1.html":
			fmt.Fprintf(w, `<script>const source='%s/parser/episode1'</script>`, "http://"+r.Host)
		case "/parser/episode1":
			fmt.Fprint(w, `<script>const source='https://media.example.test/film100-episode1.mp4'</script>`)
		case "/index.php/vod/play/id/100/sid/1/nid/2.html":
			fmt.Fprint(w, `<script>var player_aaaa={"url":"https://media.example.test/film100-episode2.mp4","encrypt":0};</script>`)
		default:
			http.NotFound(w, r)
		}
	})
	d.providerHosts[sourceYanguo] = server.URL
	items, _, err := d.fetchMaccmsCatalogPage(context.Background(), sourceYanguo, 1, "")
	if err != nil || len(items) != 2 || items[0].SourceID != "100" || items[1].SourceID != "200" {
		t.Fatalf("catalog identity: %+v %v", items, err)
	}
	drama, chapters, err := d.fetchMaccmsDetail(context.Background(), sourceYanguo, "100")
	if err != nil || drama.ID != "yanguo:100" || len(chapters) != 2 {
		t.Fatalf("chapters: %+v %+v %v", drama, chapters, err)
	}
	for i, chapter := range chapters {
		if id := yanguo171SourceID(chapter.PageURL); id != "100" {
			t.Fatalf("foreign chapter: %s", chapter.PageURL)
		}
		media, err := d.resolveDuanjuMedia(context.Background(), Task{DramaID: drama.ID, Chapter: chapter})
		want := fmt.Sprintf("https://media.example.test/film100-episode%d.mp4", i+1)
		if err != nil || media.URL != want {
			t.Fatalf("episode %d got %+v err %v", i+1, media, err)
		}
	}
	for _, path := range calls {
		if strings.Contains(path, "/id/999/") || strings.Contains(path, "/id/200/") {
			t.Fatalf("foreign movie fetched: %s", path)
		}
	}
	before := len(calls)
	chapter := chapters[0]
	chapter.PageURL = server.URL + "/index.php/vod/play/id/999/sid/1/nid/1.html"
	if _, err := d.resolveDuanjuMedia(context.Background(), Task{DramaID: drama.ID, Chapter: chapter}); err == nil {
		t.Fatal("foreign cached chapter accepted")
	}
	if len(calls) != before {
		t.Fatal("foreign chapter was fetched")
	}
}

func TestYanguo171ParserSkipsPlaceholdersAndRequiresMatchingID(t *testing.T) {
	for raw, want := range map[string]string{
		"/index.php/vod/play/id/123/sid/2/nid/8.html": "123",
		"/vodplay/456-1-2.html":                       "456",
		"/voddetail/789.html":                         "789",
		"/play?id=987":                                "987",
	} {
		if got := yanguo171SourceID(raw); got != want {
			t.Fatalf("%s => %q, want %s", raw, got, want)
		}
	}
	body := `<script>const source='https://example.test/{url}.mp4'; const source='https://media.example.test/good.mp4';</script>`
	if got := yanguo171PlayerURL(body); got != "https://media.example.test/good.mp4" {
		t.Fatalf("wrong player URL: %s", got)
	}
	values := []providerEpisode{{URL: "/vodplay/999-1-1.html"}, {URL: "/vodplay/998-1-1.html"}}
	if len(yanguo171Episodes(values, "123")) != 0 {
		t.Fatal("foreign episode list accepted")
	}
	if _, err := url.Parse(yanguo171PlayerURL(body)); err != nil {
		t.Fatal(err)
	}
}

func TestYanguo171RejectsWrongRedirect(t *testing.T) {
	d, server := duanjuFixtureDownloader(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/id/100/") {
			http.Redirect(w, r, "/index.php/vod/play/id/200/sid/1/nid/1.html", http.StatusFound)
			return
		}
		fmt.Fprint(w, `<script>const source='https://media.example.test/wrong.mp4'</script>`)
	})
	d.providerHosts[sourceYanguo] = server.URL
	page := server.URL + "/index.php/vod/play/id/100/sid/1/nid/1.html"
	if _, err := d.resolveYanguo171Page(context.Background(), page, server.URL); err == nil {
		t.Fatal("redirect to another movie accepted")
	}
}

func TestYanguo171SinglePlayerPageWithoutEpisodeLinks(t *testing.T) {
	d, server := duanjuFixtureDownloader(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.php/vod/play/id/100.html":
			fmt.Fprint(w, `<title>合成影片甲</title><div class="playlist"><a href="/index.php/vod/play/id/999/sid/1/nid/1.html">推荐影片</a></div><script>var player_aaaa={"id":"100","url":"https://media.example.test/film100.mp4","encrypt":0};</script>`)
		case "/index.php/vod/play/id/200.html":
			fmt.Fprint(w, `<title>合成影片乙</title><link rel="canonical" href="/index.php/vod/detail/id/200.html"><script>var player_aaaa={"url":"https://media.example.test/film200.mp4","encrypt":0};</script>`)
		case "/index.php/vod/play/id/500.html":
			fmt.Fprint(w, `<title>合成影片戊</title><script>var player_aaaa={"link":"/index.php/vod/play/id/500/sid/1/nid/1.html","url":"https://media.example.test/film500.mp4","encrypt":0};</script>`)
		case "/index.php/vod/play/id/300.html":
			fmt.Fprint(w, `<title>合成影片丙</title><script>var player_aaaa={"id":"999","url":"https://media.example.test/wrong.mp4","encrypt":0};</script>`)
		case "/index.php/vod/play/id/400.html":
			fmt.Fprint(w, `<title>合成影片丁</title><script>var player_aaaa={"url":"https://media.example.test/unverified.mp4","encrypt":0};</script>`)
		default:
			http.NotFound(w, r)
		}
	})
	d.providerHosts[sourceYanguo] = server.URL
	for _, id := range []string{"100", "200", "500"} {
		drama, chapters, err := d.fetchMaccmsDetail(context.Background(), sourceYanguo, id)
		if err != nil || len(chapters) != 1 || chapters[0].Title != "正片" {
			t.Fatalf("id=%s chapters=%+v err=%v", id, chapters, err)
		}
		media, err := d.resolveDuanjuMedia(context.Background(), Task{DramaID: drama.ID, Chapter: chapters[0]})
		if err != nil || media.URL != "https://media.example.test/film"+id+".mp4" {
			t.Fatalf("id=%s media=%+v err=%v", id, media, err)
		}
	}
	for _, id := range []string{"300", "400"} {
		if _, _, err := d.fetchMaccmsDetail(context.Background(), sourceYanguo, id); err == nil {
			t.Fatalf("unverified player %s accepted", id)
		}
	}
}

func TestYanguo171UnavailablePageReportsSourceUnavailable(t *testing.T) {
	d, server := duanjuFixtureDownloader(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Site Unavailable</title></head><body>Unable to access this site.</body></html>`)
	})
	d.providerHosts[sourceYanguo] = server.URL
	_, _, err := d.fetchMaccmsDetail(context.Background(), sourceYanguo, "100")
	if err == nil || !strings.Contains(err.Error(), "站源返回不可用页面") {
		t.Fatalf("wrong unavailable error: %v", err)
	}
}
