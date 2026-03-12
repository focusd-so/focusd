package usage

// Option is a function that configures a Service.
type Option func(*Service)

// WithAppBlocker configures the Service with a function to call when an application is blocked.
func WithAppBlocker(appBlocker func(appName, title, reason string, tags []string, browserURL *string)) Option {
	return func(s *Service) {
		s.appBlocker = appBlocker
	}
}
