package sqlite

import (
	"context"
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
	ctx := context.Background()
	if err := s.PutLabels(ctx, labels); err != nil {
		t.Fatalf("PutLabels: %v", err)
	}

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

	// Re-run with a shrunk sublabel list and confirm the child table
	// reflects the shrink (DELETE+INSERT semantics), and the parent
	// row count is unchanged (upsert, not duplicate insert).
	shrunk := labels[0]
	shrunk.SubLabels = shrunk.SubLabels[:2]
	if err := s.PutLabels(ctx, []model.Label{shrunk}); err != nil {
		t.Fatalf("PutLabels (rerun): %v", err)
	}
	count = scalar[int](t, s, "SELECT count(*) FROM labels")
	if count != 3 {
		t.Errorf("labels row count after rerun = %d, want 3 (upsert, not duplicate)", count)
	}
	subCount = scalar[int](t, s, "SELECT count(*) FROM label_sublabels WHERE label_id = 1")
	if subCount != 2 {
		t.Errorf("label 1 sublabel count after shrink = %d, want 2", subCount)
	}
}
