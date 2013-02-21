package catalog

import (
	"encoding/json"
	"strings"
)

// TagSet is a set of tags (strings).
type TagSet []string

// ParseTagSet builds a tag set from a comma-separated list of tags.
// Tags are trimmed of whitespace.  Any empty tags are discarded.
func ParseTagSet(s string) TagSet {
	tags := TagSet(strings.Split(s, ","))
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}
	for i := 0; i < len(tags); {
		if tags[i] == "" {
			tags = append(tags[:i], tags[i+1:]...)
		} else {
			i++
		}
	}
	return tags
}

func (ts TagSet) String() string {
	return strings.Join([]string(ts), ",")
}

func (ts TagSet) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(ts))
}

func (ts *TagSet) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*[]string)(ts))
}

// Find searches for the first occurrence of a tag in the set, or -1 if the tag is not found.
func (ts TagSet) Find(tag string) int {
	for i := range ts {
		if ts[i] == tag {
			return i
		}
	}
	return -1
}

// Has returns whether ts contains tag.
func (ts TagSet) Has(tag string) bool {
	return ts.Find(tag) != -1
}

// Unique removes all duplicate tags in the set.
func (ts *TagSet) Unique() {
	seen := make(map[string]struct{})
	for i := 0; i < len(*ts); {
		t := (*ts)[i]
		if _, found := seen[t]; found {
			*ts = append((*ts)[:i], (*ts)[i+1:]...)
		} else {
			seen[t] = struct{}{}
			i++
		}
	}
}

// Add appends a tag to the set if it is not present in the set already.
// This function returns true if the tag was added.
func (ts *TagSet) Add(tag string) (ok bool) {
	if ts.Has(tag) {
		return false
	}
	*ts = append(*ts, tag)
	return true
}

// Remove deletes all occurrences of tag in the set.
// This function returns true if the tag was found in the set.
func (ts *TagSet) Remove(tag string) (ok bool) {
	for i := 0; i < len(*ts); {
		if (*ts)[i] == tag {
			*ts = append((*ts)[:i], (*ts)[i+1:]...)
			ok = true
		} else {
			i++
		}
	}
	return
}
