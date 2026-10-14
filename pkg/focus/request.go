package focus

import (
	"errors"
	"path/filepath"

	"github.com/achandrapaul/digest/pkg/system"
)

func RequestPath(root string) string {
	return filepath.Join(root, ".state", "focus.json")
}

func requestLockPath(root string) string {
	return filepath.Join(root, ".state", "focus.lock")
}

func Write(root string, selector Selector) error {
	return system.LockedUpdate(requestLockPath(root), func() error {
		return system.WriteJSON(RequestPath(root), selector)
	})
}

func Take(root string) (Selector, bool, error) {
	var selector Selector
	var found bool
	err := system.LockedUpdate(requestLockPath(root), func() error {
		var readErr error
		found, readErr = system.ReadJSON(RequestPath(root), &selector)
		if !found && readErr == nil {
			return nil
		}
		return errors.Join(readErr, system.Remove(RequestPath(root)))
	})
	if err != nil {
		return Selector{}, false, err
	}
	return selector, found, nil
}
