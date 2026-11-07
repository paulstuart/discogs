package model

import (
	"encoding/xml"
	"testing"
)

// Confirms that MasterRef correctly decodes an element with both an
// attribute and chardata, e.g. <master_id is_main_release="true">1660109</master_id>,
// embedded inside a Release.
func TestReleaseMasterIDDecode(t *testing.T) {
	doc := `<release id="2" status="Accepted">
<title>Knockin' Boots</title>
<master_id is_main_release="true">1660109</master_id>
</release>`

	var r Release
	if err := xml.Unmarshal([]byte(doc), &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if r.ID != 2 || r.Status != "Accepted" {
		t.Fatalf("ID/Status = %d/%s, want 2/Accepted", r.ID, r.Status)
	}
	if r.MasterID == nil {
		t.Fatal("MasterID is nil, want non-nil")
	}
	if r.MasterID.ID != 1660109 {
		t.Errorf("MasterID.ID = %d, want 1660109", r.MasterID.ID)
	}
	if !r.MasterID.IsMainRelease {
		t.Errorf("MasterID.IsMainRelease = false, want true")
	}
}

func TestReleaseMasterIDAbsent(t *testing.T) {
	doc := `<release id="3" status="Accepted"><title>No Master</title></release>`
	var r Release
	if err := xml.Unmarshal([]byte(doc), &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if r.MasterID != nil {
		t.Errorf("MasterID = %+v, want nil when element absent", r.MasterID)
	}
}
