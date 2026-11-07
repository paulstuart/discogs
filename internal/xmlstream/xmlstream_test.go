package xmlstream

import (
	"errors"
	"strings"
	"testing"
)

type item struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:"name"`
}

func TestStreamDecode(t *testing.T) {
	doc := `<items>
<item id="1"><name>alpha</name></item>
<item id="2"><name>beta</name></item>
<item id="3"><name>gamma</name></item>
</items>`

	var got []item
	err := StreamDecode(strings.NewReader(doc), "item", func(it item) error {
		got = append(got, it)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamDecode: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d items, want 3", len(got))
	}
	want := []item{{1, "alpha"}, {2, "beta"}, {3, "gamma"}}
	for i, it := range got {
		if it != want[i] {
			t.Errorf("item %d = %+v, want %+v", i, it, want[i])
		}
	}
}

func TestStreamDecodeCallbackError(t *testing.T) {
	doc := `<items><item id="1"><name>alpha</name></item><item id="2"><name>beta</name></item></items>`
	sentinel := errors.New("stop")
	count := 0
	err := StreamDecode(strings.NewReader(doc), "item", func(it item) error {
		count++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
	if count != 1 {
		t.Fatalf("callback invoked %d times, want 1 (should stop on first error)", count)
	}
}

func TestStreamDecodeIgnoresOtherElements(t *testing.T) {
	doc := `<items><other id="9"/><item id="1"><name>alpha</name></item></items>`
	var got []item
	err := StreamDecode(strings.NewReader(doc), "item", func(it item) error {
		got = append(got, it)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamDecode: %v", err)
	}
	if len(got) != 1 || got[0].Name != "alpha" {
		t.Fatalf("got %+v, want single alpha item", got)
	}
}
