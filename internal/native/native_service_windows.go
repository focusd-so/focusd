//go:build windows
// +build windows

package native

// NativeService is a Wails-bound service for managing OS permissions.
// On Windows, these are unimplemented stubs.
type NativeService struct {
	started bool
}

func NewNativeService() *NativeService {
	return &NativeService{}
}

func (s *NativeService) CheckAccessibility() bool {
	return true
}

func (s *NativeService) RequestAccessibility() bool {
	return true
}

func (s *NativeService) CheckAutomation(appID string) bool {
	return true
}

func (s *NativeService) RequestAutomation(appID string) bool {
	return true
}

func (s *NativeService) OpenSettings() {
}

func (s *NativeService) StartObserver() {
	once.Do(func() {
		go startObserver()
	})
}

func (s *NativeService) EnableLoginItem() error {
	return nil
}

func (s *NativeService) DisableLoginItem() error {
	return nil
}

func (s *NativeService) LoginItemEnabled() bool {
	return false
}

func (s *NativeService) GetInstalledBrowsers() []InstalledBrowser {
	return []InstalledBrowser{}
}
