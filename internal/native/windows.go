//go:build windows
// +build windows

package native

import "context"

func OnActiveAppTitleChange(_ context.Context, fn func(event NativeEvent)) {

}

func GetIdentity() (string, error) {
	return "", nil
}

func BlockURL(url string, title string, reason string, tags string, bundleID string) error {
	return nil
}

func BlockApp(name string) error {
	return nil
}

func MinimiseApp(bundleID string) error {
	return nil
}
