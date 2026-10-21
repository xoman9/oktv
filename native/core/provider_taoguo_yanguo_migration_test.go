package core

import "testing"

func TestZhenguoAddedMacCMSSources(t *testing.T) {
 for _, tc := range []struct{id, host, name string}{
  {sourceTaoguo, "lujj31.buzz", "桃果"},
  {sourceYanguo, "xqxq1.cc", "艷果"},
 } {
  spec, ok := duanjuSourceSpecFor(tc.id)
  if !ok || spec.Name != tc.name || spec.Kind != "maccms" {t.Fatalf("%s registration: %+v %v",tc.id,spec,ok)}
  if duanjuSourceForHost(tc.host) != tc.id {t.Fatalf("%s host routing failed",tc.id)}
  if !duanjuSupportsSearch(tc.id) || !duanjuSupportsPaging(tc.id) {t.Fatalf("%s missing capabilities",tc.id)}
  if len(maccmsDetailCandidates(tc.id,"https://example.com","123")) == 0 {t.Fatalf("%s missing detail paths",tc.id)}
 }
}
