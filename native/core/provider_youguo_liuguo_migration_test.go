package core

import "testing"

func TestZhenguoYouguoLiuguoRegistration(t *testing.T) {
 for _, tc := range []struct{id, host, name string}{
  {sourceYouguo, "youavhub.com", "柚果 🔞"},
  {sourceLiuguo, "llsp.me", "榴果 🔞"},
 } {
  spec, ok := duanjuSourceSpecFor(tc.id)
  if !ok || spec.Name != tc.name || spec.Kind != "maccms" { t.Fatalf("%s registration invalid: %+v %v", tc.id, spec, ok) }
  if duanjuSourceForHost(tc.host) != tc.id { t.Fatalf("%s host routing failed", tc.id) }
  if !duanjuSupportsSearch(tc.id) || !duanjuSupportsPaging(tc.id) { t.Fatalf("%s capability flags invalid", tc.id) }
  if len(maccmsDetailCandidates(tc.id, "https://example.com", "123")) == 0 { t.Fatalf("%s detail routes missing", tc.id) }
 }
}
