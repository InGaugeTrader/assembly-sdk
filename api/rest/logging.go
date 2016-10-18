package rest

type Logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

func (s *Server) debugf(fmt string, args ...interface{}) {
	if s.options.logger != nil {
		s.options.logger.Debugf(fmt, args...)
	}
}

func (s *Server) infof(fmt string, args ...interface{}) {
	if s.options.logger != nil {
		s.options.logger.Infof(fmt, args...)
	}
}

func (s *Server) warnf(fmt string, args ...interface{}) {
	if s.options.logger != nil {
		s.options.logger.Warnf(fmt, args...)
	}
}

func (s *Server) errorf(fmt string, args ...interface{}) {
	if s.options.logger != nil {
		s.options.logger.Errorf(fmt, args...)
	}
}
