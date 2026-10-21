package core

import "testing"

// The migration must remain additive: register the new source without
// changing any of the existing source identities or playback routes.
func TestMiguoSourceRegistration(t *testing.T) {
 spec, ok := duanjuSourceSpecFor(sourceMiguo)
 if !ok || spec.Name != "米果" || spec.Kind != "maccms" {
  t.Fatalf("miguo source not registered: %+v found=%v", spec, ok)
 }
 if !duanjuSupportsSearch(sourceMiguo) || !duanjuSupportsPaging(sourceMiguo) {
  t.Fatal("miguo source must support search and catalog paging")
 }
 if source := canonicalProviderSource("miguo"); source != sourceMiguo {
  t.Fatalf("canonical source unexpected: %q", source)
 }
 if source := duanjuSourceForHost("dmxq40.com"); source != sourceMiguo {
  t.Fatalf("host lookup unexpected: %q", source)
 }
}
