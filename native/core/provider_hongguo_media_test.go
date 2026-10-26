package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestHongguoXanerixNativeMedia(t *testing.T) {
	model := map[string]any{"video_duration": 12, "video_list": map[string]any{"v1": map[string]any{"main_url": "https://media.example.test/episode.mp4", "video_meta": map[string]any{"vheight": 720, "codec_type": "h264"}}}}
	encoded, _ := json.Marshal(model)
	for _, value := range []any{model, string(encoded), base64.StdEncoding.EncodeToString(encoded)} {
		engine := sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/novel/player/video_model/v1/" || r.Method != http.MethodPost {
				t.Fatalf("unexpected donor route: %s", r.URL)
			}
			for _, key := range []string{"X-Gorgon", "X-Khronos", "X-Argus", "X-Ladon"} {
				if r.Header.Get(key) == "" {
					t.Fatalf("missing signature: %s", key)
				}
			}
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			if json.Unmarshal(body, &payload) != nil || payload["video_id"] != "800001" || mapInt(payload, "content_type") != 1 || mapInt(nestedMap(payload, "biz_param"), "video_platform") != 3 {
				t.Fatalf("wrong donor body: %s", body)
			}
			response, _ := json.Marshal(map[string]any{"code": 0, "data": map[string]any{"video_model": value}})
			return sourceFixtureResponse(r, 200, string(response)), nil
		})
		media, err := engine.downloader.resolveProviderMedia(context.Background(), Task{DramaID: providerDramaID(sourceHongguo, "700001"), Chapter: Chapter{Source: sourceHongguo, VideoURL: "hongguo-cenc://800001"}})
		if err != nil || media.URL != "https://media.example.test/episode.mp4" || media.Quality != 720 {
			t.Fatalf("media=%+v err=%v", media, err)
		}
	}
}
