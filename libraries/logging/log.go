package logging

import (
	"fmt"

	"github.com/google/deck"
)

type Message struct {
	Prefix    string
	Format    string
	Vs        []any
	Attribute deck.Attrib
}

func (m Message) Infof() {
	deck.InfofA(fmt.Sprintf("[%s]: %s", m.Prefix, fmt.Sprintf(m.Format, m.Vs...))).With(m.Attribute).Go()
}

func (m Message) Warnf() {
	deck.WarningfA(fmt.Sprintf("[%s]: %s", m.Prefix, fmt.Sprintf(m.Format, m.Vs...))).With(m.Attribute).Go()
}

func (m Message) Errorf() {
	deck.ErrorfA(fmt.Sprintf("[%s]: %s", m.Prefix, fmt.Sprintf(m.Format, m.Vs...))).With(m.Attribute).Go()
}
