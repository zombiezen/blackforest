package catalog

import (
	"encoding/json"
	"os"
)

const permMask os.FileMode = 0666

func readJSON(path string, v interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}

func writeJSON(path string, v interface{}, excl bool) (retErr error) {
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if excl {
		flag |= os.O_EXCL
	}
	f, err := os.OpenFile(path, flag, permMask)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil && retErr == nil {
			retErr = err
		}
	}()
	if err := json.NewEncoder(f).Encode(v); err != nil {
		return err
	}
	return nil
}
