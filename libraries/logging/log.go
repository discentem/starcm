package logging

import (
	"fmt"

	"github.com/google/deck"
)

func Infof(format string, v ...any) {
	Log("Infof", nil, "info", format, v...)
}

func Log(prefix string, attribute deck.Attrib, level string, format string, v ...any) {
	m := message{
		Prefix: prefix,
		Format: format,
		Vs:     &v,
	}
	if attribute == nil {
		m.Attribute = func(*deck.AttribStore) {}
	} else {
		m.Attribute = attribute
	}
	switch level {
	case "info":
		m.Infof()
	case "warn":
		m.Warnf()
	case "error":
		m.Errorf()
	default:
		m.Infof()
	}
}

type message struct {
	Prefix    string
	Format    string
	Vs        *[]any
	Attribute func(*deck.AttribStore)
}

func (m message) Text() string {
	if m.Vs == nil {
		return fmt.Sprintf("[%s]: %s", m.Prefix, m.Format)
	}
	return fmt.Sprintf("[%s]: %v", m.Prefix, fmt.Sprintf(m.Format, *m.Vs...))
}

// Infof, Warnf and Errorf pass the already formatted Text as an argument, so
// a % in a label or path isn't read as a verb.
func (m message) Infof() {
	deck.InfofA("%s", m.Text()).With(m.Attribute).Go()
}

func (m message) Warnf() {
	deck.WarningfA("%s", m.Text()).With(m.Attribute).Go()
}

func (m message) Errorf() {
	deck.ErrorfA("%s", m.Text()).With(m.Attribute).Go()
}
