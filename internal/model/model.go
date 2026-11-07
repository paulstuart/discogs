// Package model defines the canonical Go representation of Discogs dump
// records. Every struct carries both xml tags (for streaming decode of the
// data dumps) and json tags (for potential JSON output), so this is the one
// set of types used by both the XML decoder and the store layer.
package model

// NameRef is a reference to another artist by id, used for an artist's
// aliases and group members: <name id="N">Name</name>.
type NameRef struct {
	ID   int64  `xml:"id,attr" json:"id"`
	Name string `xml:",chardata" json:"name"`
}

// Artist is a <artist> record from artists.xml.
type Artist struct {
	ID             int64     `xml:"id" json:"id"`
	Name           string    `xml:"name" json:"name"`
	RealName       string    `xml:"realname" json:"real_name,omitempty"`
	Profile        string    `xml:"profile" json:"profile,omitempty"`
	DataQuality    string    `xml:"data_quality" json:"data_quality,omitempty"`
	URLs           []string  `xml:"urls>url" json:"urls,omitempty"`
	NameVariations []string  `xml:"namevariations>name" json:"name_variations,omitempty"`
	Aliases        []NameRef `xml:"aliases>name" json:"aliases,omitempty"`
	Members        []NameRef `xml:"members>name" json:"members,omitempty"`
}

// LabelRef is a reference to another label by id, used for sublabels and
// parent labels: <label id="N">Name</label>.
type LabelRef struct {
	ID   int64  `xml:"id,attr" json:"id"`
	Name string `xml:",chardata" json:"name"`
}

// Label is a <label> record from labels.xml.
type Label struct {
	ID          int64      `xml:"id" json:"id"`
	Name        string     `xml:"name" json:"name"`
	ContactInfo string     `xml:"contactinfo" json:"contact_info,omitempty"`
	Profile     string     `xml:"profile" json:"profile,omitempty"`
	DataQuality string     `xml:"data_quality" json:"data_quality,omitempty"`
	URLs        []string   `xml:"urls>url" json:"urls,omitempty"`
	SubLabels   []LabelRef `xml:"sublabels>label" json:"sub_labels,omitempty"`
	ParentLabel *LabelRef  `xml:"parentLabel" json:"parent_label,omitempty"`
}

// ArtistCredit is an artist credit on a master, release, or track:
// <artist><id>..</id><name>..</name><anv>..</anv><join>..</join><role>..</role></artist>.
type ArtistCredit struct {
	ID   int64  `xml:"id" json:"id"`
	Name string `xml:"name" json:"name"`
	ANV  string `xml:"anv" json:"anv,omitempty"`
	Join string `xml:"join" json:"join,omitempty"`
	Role string `xml:"role" json:"role,omitempty"`
}

// Video is a <video> element attached to a master or release.
type Video struct {
	Src         string `xml:"src,attr" json:"src"`
	Duration    int    `xml:"duration,attr" json:"duration,omitempty"`
	Embed       bool   `xml:"embed,attr" json:"embed,omitempty"`
	Title       string `xml:"title" json:"title,omitempty"`
	Description string `xml:"description" json:"description,omitempty"`
}

// Master is a <master> record from masters.xml. Unlike Artist/Label, id is
// an XML attribute rather than a child element.
type Master struct {
	ID          int64          `xml:"id,attr" json:"id"`
	MainRelease int64          `xml:"main_release" json:"main_release"`
	Artists     []ArtistCredit `xml:"artists>artist" json:"artists,omitempty"`
	Genres      []string       `xml:"genres>genre" json:"genres,omitempty"`
	Styles      []string       `xml:"styles>style" json:"styles,omitempty"`
	Year        int            `xml:"year" json:"year,omitempty"`
	Title       string         `xml:"title" json:"title"`
	DataQuality string         `xml:"data_quality" json:"data_quality,omitempty"`
	Videos      []Video        `xml:"videos>video" json:"videos,omitempty"`
}

// MasterRef captures <master_id is_main_release="true">123</master_id>.
type MasterRef struct {
	IsMainRelease bool  `xml:"is_main_release,attr" json:"is_main_release"`
	ID            int64 `xml:",chardata" json:"id"`
}

// ReleaseLabel is a label credit on a release (not full label data):
// <label name=".." catno=".." id="N"/>.
type ReleaseLabel struct {
	Name  string `xml:"name,attr" json:"name"`
	Catno string `xml:"catno,attr" json:"catno,omitempty"`
	ID    int64  `xml:"id,attr" json:"id"`
}

// ExtraArtist is a non-primary artist credit on a release, e.g. producer or
// writer. Role is freeform and may contain multiple comma-separated roles.
type ExtraArtist struct {
	ID   int64  `xml:"id" json:"id"`
	Name string `xml:"name" json:"name"`
	ANV  string `xml:"anv" json:"anv,omitempty"`
	Role string `xml:"role" json:"role,omitempty"`
}

// Format is a release format, e.g. Vinyl/CD, with free-text descriptions.
type Format struct {
	Name         string   `xml:"name,attr" json:"name"`
	Qty          string   `xml:"qty,attr" json:"qty,omitempty"`
	Text         string   `xml:"text,attr" json:"text,omitempty"`
	Descriptions []string `xml:"descriptions>description" json:"descriptions,omitempty"`
}

// Track is one entry in a release's tracklist. Position is freeform
// (e.g. "A", "B1"), not strictly numeric.
type Track struct {
	Position string         `xml:"position" json:"position"`
	Title    string         `xml:"title" json:"title"`
	Duration string         `xml:"duration" json:"duration,omitempty"`
	Artists  []ArtistCredit `xml:"artists>artist" json:"artists,omitempty"`
}

// Identifier is a self-closing <identifier type=".." description=".." value=".."/>.
type Identifier struct {
	Type        string `xml:"type,attr" json:"type"`
	Description string `xml:"description,attr" json:"description,omitempty"`
	Value       string `xml:"value,attr" json:"value"`
}

// Company is a credited company on a release, e.g. "Pressed By".
type Company struct {
	ID             int64  `xml:"id" json:"id"`
	Name           string `xml:"name" json:"name"`
	EntityType     int64  `xml:"entity_type" json:"entity_type,omitempty"`
	EntityTypeName string `xml:"entity_type_name" json:"entity_type_name,omitempty"`
}

// Release is a <release> record from releases.xml. Both id and status are
// XML attributes.
type Release struct {
	ID           int64          `xml:"id,attr" json:"id"`
	Status       string         `xml:"status,attr" json:"status,omitempty"`
	Title        string         `xml:"title" json:"title"`
	Artists      []ArtistCredit `xml:"artists>artist" json:"artists,omitempty"`
	ExtraArtists []ExtraArtist  `xml:"extraartists>artist" json:"extra_artists,omitempty"`
	Labels       []ReleaseLabel `xml:"labels>label" json:"labels,omitempty"`
	Formats      []Format       `xml:"formats>format" json:"formats,omitempty"`
	Genres       []string       `xml:"genres>genre" json:"genres,omitempty"`
	Styles       []string       `xml:"styles>style" json:"styles,omitempty"`
	Country      string         `xml:"country" json:"country,omitempty"`
	// Released is kept as a raw string (e.g. "1999-03-00") rather than
	// parsed into a date type, since Discogs allows partial/zero dates.
	Released    string       `xml:"released" json:"released,omitempty"`
	Notes       string       `xml:"notes" json:"notes,omitempty"`
	MasterID    *MasterRef   `xml:"master_id" json:"master_id,omitempty"`
	DataQuality string       `xml:"data_quality" json:"data_quality,omitempty"`
	Tracklist   []Track      `xml:"tracklist>track" json:"tracklist,omitempty"`
	Identifiers []Identifier `xml:"identifiers>identifier" json:"identifiers,omitempty"`
	Videos      []Video      `xml:"videos>video" json:"videos,omitempty"`
	Companies   []Company    `xml:"companies>company" json:"companies,omitempty"`
}
