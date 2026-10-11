package clipboard

import "github.com/atotto/clipboard"

var writeAll = clipboard.WriteAll

func Copy(text string) error {
	if text == "" {
		return nil
	}
	return writeAll(text)
}
