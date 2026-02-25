//go:build linux
// +build linux

package native

import "context"

func OnActiveAppTitleChange(_ context.Context, fn func(event NativeEvent)) {

}

func GetIdentity() (string, error) {
	return "", nil
}

func BlockURL(targetURL, title, reason string, tags []string, appName string) error {
	return nil
}

func BlockApp(appName, title, reason string, tags []string) error {
	return nil
}

var (
	onTitleChange func(event NativeEvent)
	onIdleChange  func(idleSeconds float64)
)

func OnTitleChange(callback func(event NativeEvent)) {
	onTitleChange = callback
}

func OnIdleChange(callback func(idleSeconds float64)) {
	onIdleChange = callback
}

func StartObserver() {

}
