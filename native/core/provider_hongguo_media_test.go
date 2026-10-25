package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHongguo0716NativeRequestAndFallback(t *testing.T) {
	engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/novel/player/multi_video_model/v1/":
			for _, key := range []string{"X-Gorgon", "X-Khronos", "X-Ladon", "X-Argus", "X-Helios", "X-Medusa"} {
				if r.Header.Get(key) == "" {
					t.Fatalf("missing donor signature: %s", key)
				}
			}
			for key, want := range map[string]string{"host_abi": "arm64-v8a", "dragon_device_type": "phone", "pv_player": "73532", "player_so_load": "1"} {
				if r.URL.Query().Get(key) != want {
					t.Fatalf("query %s=%s", key, r.URL.Query().Get(key))
				}
			}
			if r.Method != http.MethodPost || r.Header.Get("Passport-Sdk-Version") != "50561" {
				t.Fatal("donor request headers missing")
			}
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			if json.Unmarshal(body, &payload) != nil || !strings.Contains(string(body), `"1004":["800001"]`) || payload["dr_scene"] != nil {
				t.Fatalf("wrong donor payload: %s", body)
			}
			return sourceFixtureResponse(r, 200, `{"data":{"800001":{"video_model":"{\"fallback_api\":\"https://media.fqnovel.com/fallback\"}"}}}`), nil
		case "/fallback":
			return sourceFixtureResponse(r, 200, `{"video_info":{"data":{"video_duration":12,"video_list":{"v1":{"main_url":"https://media.example.test/episode.mp4","video_meta":{"vheight":720,"codec_type":"h264"}}}}}}`), nil
		default:
			t.Fatalf("unexpected donor route: %s", r.URL)
			return nil, nil
		}
	})
	media, err := engine.downloader.resolveProviderMedia(context.Background(), Task{DramaID: providerDramaID(sourceHongguo, "700001"), Chapter: Chapter{Source: sourceHongguo, VideoURL: "hongguo-cenc://800001"}})
	if err != nil {
		t.Fatal(err)
	}
	if media.URL != "https://media.example.test/episode.mp4" || media.Quality != 720 {
		t.Fatalf("wrong media: %+v", media)
	}
}

func TestHongguo0716VideoModelAndFallbackShapes(t *testing.T) {
	for _, value := range []any{`{"video_duration":12}`, map[string]any{"video_duration": json.Number("12")}} {
		if model := hongguoVideoModel(value); mapString(model, "video_duration") != "12" {
			t.Fatal(model)
		}
	}
	if len(hongguoVideoModel("invalid")) != 0 {
		t.Fatal("invalid model accepted")
	}
	for _, value := range []any{"https://media.fqnovel.com/fallback", `{"fallback_api":"https://media.fqnovel.com/fallback"}`, []any{"https://media.fqnovel.com/fallback"}, map[string]any{"fallback_api": "https://media.fqnovel.com/fallback"}} {
		if got := hongguoFallbackAddress(value); got != "https://media.fqnovel.com/fallback" {
			t.Fatal(got)
		}
	}
}
