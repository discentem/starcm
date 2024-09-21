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

func (m message) Infof() {
	if m.Attribute == nil {
		deck.Infof(m.Text())
	}
	deck.InfofA(m.Text()).With(m.Attribute).Go()
}

func (m message) Warnf() {
	if m.Attribute == nil {
		deck.Warningf(m.Text())
	}

	deck.WarningfA(m.Text()).With(m.Attribute).Go()
}

func (m message) Errorf() {
	if m.Attribute == nil {
		deck.Errorf(m.Text())
	}

	deck.ErrorfA(m.Text()).With(m.Attribute).Go()
}
