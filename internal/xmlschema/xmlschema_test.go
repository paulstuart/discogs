package xmlschema

import (
	"os"
	"testing"
)

func TestSniffReleasesFixture(t *testing.T) {
	f, err := os.Open("../store/sqlite/testdata/releases_fixture.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rep, err := Sniff(f)
	if err != nil {
		t.Fatal(err)
	}

	release, ok := rep.Paths["releases/release"]
	if !ok {
		t.Fatal("expected releases/release path")
	}
	if release.Count != 1 {
		t.Errorf("releases/release count = %d, want 1", release.Count)
	}
	if _, ok := release.Attrs["id"]; !ok {
		t.Error(`releases/release missing attr "id"`)
	}
	if _, ok := release.Attrs["status"]; ok {
		t.Error(`releases/release has "status" attr, but fixture does not set it`)
	}

	title, ok := rep.Paths["releases/release/title"]
	if !ok || !title.HasText {
		t.Error("expected releases/release/title to have text content")
	}

	video, ok := rep.Paths["releases/release/videos/video"]
	if !ok {
		t.Fatal("expected releases/release/videos/video path")
	}
	for _, attr := range []string{"src", "duration", "embed"} {
		if _, ok := video.Attrs[attr]; !ok {
			t.Errorf("releases/release/videos/video missing attr %q", attr)
		}
	}
}
