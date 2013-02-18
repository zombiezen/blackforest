package catalog

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

const (
	IDSize       = 9
	IDEncodedLen = 12
)

var idEncoding = base64.URLEncoding

// An ID is a 72-bit unsigned integer that is used to identify entities in a catalog.
type ID [IDSize]byte

// GenerateID builds a new ID from a random number generator.
func GenerateID() (ID, error) {
	var id ID
	_, err := io.ReadFull(rand.Reader, id[:])
	return id, err
}

func (id ID) String() string {
	return idEncoding.EncodeToString(id[:])
}

func (id ID) MarshalJSON() ([]byte, error) {
	data := make([]byte, IDEncodedLen+2)
	data[0], data[len(data)-1] = '"', '"'
	idEncoding.Encode(data[1:len(data)-1], id[:])
	return data, nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	n := len(data)
	if n == 0 || data[0] != '"' || data[n-1] != '"' {
		return errors.New("attempt to unmarshal non-string ID from JSON")
	} else if n != IDEncodedLen+2 {
		return errors.New("JSON ID has wrong size")
	}
	_, err := idEncoding.Decode((*id)[:], data[1:n-1])
	return err
}
