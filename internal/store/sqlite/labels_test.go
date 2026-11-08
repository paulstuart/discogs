package sqlite

import (
	"os"
	"testing"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/xmlstream"
)

func loadLabelsFixture(t *testing.T) []model.Label {
	t.Helper()
	f, err := os.Open("testdata/labels_fixture.xml")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()
	var labels []model.Label
	if err := xmlstream.StreamDecode(f, "label", func(l model.Label) error {
		labels = append(labels, l)
		return nil
	}); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return labels
}

func TestPutLabels(t *testing.T) {
	labels := loadLabelsFixture(t)
	if len(labels) != 3 {
		t.Fatalf("got %d labels from fixture, want 3", len(labels))
	}

	s := openTestStore(t)
	putLabelsFull(t, s, labels)

	count := scalar[int](t, s, "SELECT count(*) FROM labels")
	if count != 3 {
		t.Errorf("labels row count = %d, want 3", count)
	}

	name := scalar[string](t, s, "SELECT name FROM labels WHERE id = 1")
	if name != "Planet E" {
		t.Errorf("label 1 name = %q, want Planet E", name)
	}

	subCount := scalar[int](t, s, "SELECT count(*) FROM label_sublabels WHERE label_id = 1")
	if subCount != 8 {
		t.Errorf("label 1 sublabel count = %d, want 8", subCount)
	}

	// A second load (e.g. next month's dump) with a shrunk sublabel list
	// should reflect the shrink: PrepareLabels wipes the tables, so the
	// reload is a full replace rather than a merge with the stale rows.
	labels[0].SubLabels = labels[0].SubLabels[:2]
	putLabelsFull(t, s, labels)
	count = scalar[int](t, s, "SELECT count(*) FROM labels")
	if count != 3 {
		t.Errorf("labels row count after rerun = %d, want 3 (upsert, not duplicate)", count)
	}
	subCount = scalar[int](t, s, "SELECT count(*) FROM label_sublabels WHERE label_id = 1")
	if subCount != 2 {
		t.Errorf("label 1 sublabel count after shrink = %d, want 2", subCount)
	}
}
