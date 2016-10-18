package scanner

// Logger is an interface wrapping logging calls required by the scanner.
type Logger interface {
	Infof(string, ...interface{})
}

// infof wraps info level logging calls from the scanner. If a logger is
// provided, the message is sent there, otherwise ignored.
func (s *Scanner) infof(format string, args ...interface{}) {
	if s.options.logger != nil {
		s.options.logger.Infof(format, args...)
	}
}
