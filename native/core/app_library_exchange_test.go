package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func exchangeFixtureEngine(t *testing.T) *nativeEngine {
	previous := buildAllSources
	buildAllSources = "true"
	t.Cleanup(func() { buildAllSources = previous })
	return sourceFixtureEngine(t, func(r *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("unexpected network request: %s", r.URL)
	})
}

func TestLibraryExchangePaginationRoundTripAndResume(t *testing.T) {
	original := exchangeFixtureEngine(t)
	source := sourceLiuguo
	drama := nativeDrama{ID: providerDramaID(source, "101"), Source: source, SourceID: "101", Title: "合成影片"}
	original.catalogs[source] = []nativeDrama{drama}
	original.catalogs[source+"|23"] = []nativeDrama{drama}
	original.catalogStates[source] = nativeCatalogState{Page: 7, HasMore: true, UpdatedAt: time.Now()}
	original.catalogStates[source+"|23"] = nativeCatalogState{Page: 12, HasMore: false, UpdatedAt: time.Now()}
	payload, _, err := original.nativeLibraryExportXZ()
	if err != nil {
		t.Fatal(err)
	}
	document, err := nativeLibraryImportXZ(payload)
	if err != nil {
		t.Fatal(err)
	}
	target := exchangeFixtureEngine(t)
	if _, err = target.nativeLibraryImport(document); err != nil {
		t.Fatal(err)
	}
	restored := exchangeFixtureEngine(t)
	restored.directory = target.directory
	restored.loadCatalogCache()
	if got := restored.nativeCached(source); got.Page != 7 || !got.HasMore || len(got.Items) != 1 {
		t.Fatalf("source state lost: %+v", got)
	}
	if got := restored.nativeCached(source + "|23"); got.Page != 12 || got.HasMore || len(got.Items) != 1 {
		t.Fatalf("category state lost: %+v", got)
	}
	var paths []string
	restored.downloader.client.Transport = sourceFixtureTransport(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		return sourceFixtureResponse(r, 200, `<div class="video-item"><a href="/voddetail/102.html" title="合成影片乙">合成影片乙</a></div>`), nil
	})
	if err := restored.loadMoreSourceCatalogPages(context.Background(), source, 1); err != nil {
		t.Fatal(err)
	}
	if strings.Join(paths, ",") != "/vodtype/3-8.html" {
		t.Fatal(paths)
	}
}

func TestLibraryExchangePaginationDoesNotRegressAndSavesStateOnly(t *testing.T) {
	engine := exchangeFixtureEngine(t)
	source := sourceLiuguo
	drama := nativeDrama{ID: providerDramaID(source, "101"), Source: source, SourceID: "101", Title: "合成影片", MetadataSchema: 1}
	engine.catalogs[source] = []nativeDrama{drama}
	engine.catalogStates[source] = nativeCatalogState{Page: 9, HasMore: true}
	entry, _ := nativeLibraryExchangeFromDrama(drama)
	document := nativeLibraryExchange{Version: 1, Dramas: []nativeLibraryExchangeDrama{entry}, Catalogs: map[string]nativeLibraryExchangeCatalog{source: {Page: 7, HasMore: false, IDs: []string{drama.ID}}}}
	if _, err := engine.nativeLibraryImport(document); err != nil {
		t.Fatal(err)
	}
	if engine.catalogStates[source].Page != 9 || !engine.catalogStates[source].HasMore {
		t.Fatal("older import regressed cursor")
	}
	document.Catalogs[source] = nativeLibraryExchangeCatalog{Page: 9, HasMore: false, IDs: []string{drama.ID}}
	if _, err := engine.nativeLibraryImport(document); err != nil {
		t.Fatal(err)
	}
	restored := exchangeFixtureEngine(t)
	restored.directory = engine.directory
	restored.loadCatalogCache()
	if restored.catalogStates[source].Page != 9 || restored.catalogStates[source].HasMore {
		t.Fatal("state-only change was not persisted")
	}
	if err := restored.loadMoreSourceCatalogPages(context.Background(), source, 1); err != nil {
		t.Fatal("completed source fetched another page", err)
	}
}

func TestLibraryExchangeLegacyAndInvalidProgress(t *testing.T) {
	engine := exchangeFixtureEngine(t)
	entry := nativeLibraryExchangeDrama{ID: providerDramaID(sourceLiuguo, "101"), Title: "合成影片"}
	document := nativeLibraryExchange{Version: 1, Dramas: []nativeLibraryExchangeDrama{entry}}
	if _, err := engine.nativeLibraryImport(document); err != nil {
		t.Fatal(err)
	}
	if engine.catalogStates[sourceLiuguo].Page != 1 || !engine.catalogStates[sourceLiuguo].HasMore {
		t.Fatal("legacy import changed")
	}
	document.Catalogs = map[string]nativeLibraryExchangeCatalog{sourceLiuguo: {Page: 99, IDs: []string{"missing"}}}
	if _, err := engine.nativeLibraryImport(document); err != nil {
		t.Fatal(err)
	}
	if engine.catalogStates[sourceLiuguo].Page != 1 {
		t.Fatal("invalid cursor accepted")
	}
}

func TestLibraryExchangeHongguoCursorRestoredOnFreshImport(t *testing.T) {
	original := exchangeFixtureEngine(t)
	drama := nativeDrama{ID: providerDramaID(sourceHongguo, "101"), Source: sourceHongguo, SourceID: "101", Title: "合成影片"}
	original.catalogs[sourceHongguo] = []nativeDrama{drama}
	original.catalogStates[sourceHongguo] = nativeCatalogState{Page: 8, HasMore: true}
	key := hongguoAppGenres[0].key
	original.hongguoCatalog = &hongguoCatalogState{Version: 1, DeviceID: "123456789", InstallID: "987654321", Feeds: map[string]hongguoCatalogCursor{key: {Offset: 80, Initialized: true}}}
	target := exchangeFixtureEngine(t)
	if _, err := target.nativeLibraryImport(original.nativeLibraryExport()); err != nil {
		t.Fatal(err)
	}
	if got := target.downloader.hongguoCatalogSnapshot().Feeds[key]; !got.Initialized || got.Offset != 80 {
		t.Fatal("native cursor lost", got)
	}
	restored := exchangeFixtureEngine(t)
	restored.directory = target.directory
	restored.loadCatalogCache()
	if restored.downloader.hongguoCatalogSnapshot().Feeds[key].Offset != 80 {
		t.Fatal("native cursor not persisted")
	}
}

func TestLibraryExchangeLargeXZIncludesPagination(t *testing.T) {
	original := exchangeFixtureEngine(t)
	id := providerDramaID(sourceLiuguo, "101")
	original.catalogs[sourceLiuguo] = []nativeDrama{{ID: id, Source: sourceLiuguo, SourceID: "101", Title: "synthetic"}}
	original.catalogStates[sourceLiuguo] = nativeCatalogState{Page: 7, HasMore: true}
	document := original.nativeLibraryExport()
	body, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	body = append(body[:len(body)-1], []byte(",\"padding\":\""+strings.Repeat("x", 33<<20)+"\"}")...)
	compressed, err := nativeLibraryExchangeCompressPure(body)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := nativeLibraryImportXZ(base64.StdEncoding.EncodeToString(compressed))
	if err != nil {
		t.Fatal(err)
	}
	target := exchangeFixtureEngine(t)
	if _, err := target.nativeLibraryImport(decoded); err != nil {
		t.Fatal(err)
	}
	if target.catalogStates[sourceLiuguo].Page != 7 || !target.catalogStates[sourceLiuguo].HasMore {
		t.Fatal("pagination not restored")
	}
}
