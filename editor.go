package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// runEditor opens the user's editor populated with text and returns the modified text.
func runEditor(text string) (string, error) {
	const tempFilePrefix = "blackforest-editor-"

	f, err := ioutil.TempFile("", tempFilePrefix)
	if err != nil {
		return text, err
	}
	// TODO(light): log errors?
	defer f.Close()
	defer os.Remove(f.Name())

	if _, err := io.WriteString(f, text); err != nil {
		return text, err
	}

	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		// TODO(light): flags
		c = exec.Command(editor, f.Name())
	} else {
		c = exec.Command("sh", "-c", editor+" "+shellEscape(f.Name()))
	}
	c.Stderr = os.Stderr
	cerr := c.Run()
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return text, err
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return text, err
	}
	return string(data), cerr
}

// shellEscape returns a string that surrounds arg with single quotes and
// escapes any single quotes inside arg by using double quotes.
func shellEscape(arg string) string {
	parts := make([]string, 0, 1)
	for {
		if i := strings.IndexRune(arg, '\''); i != -1 {
			parts, arg = append(parts, arg[:i]), arg[i+1:]
		} else {
			break
		}
	}
	if len(arg) > 0 {
		parts = append(parts, arg)
	}
	return "'" + strings.Join(parts, `'"'"'`) + "'"
}

// windowsEscape returns a string that surrounds arg with double quotes and
// escapes any double quotes inside arg by doubling them.
func windowsEscape(arg string) string {
	parts := make([]string, 0, 1)
	for {
		if i := strings.IndexRune(arg, '"'); i != -1 {
			parts, arg = append(parts, arg[:i]), arg[i+1:]
		} else {
			break
		}
	}
	if len(arg) > 0 {
		parts = append(parts, arg)
	}
	return `"` + strings.Join(parts, `""`) + `"`
}
