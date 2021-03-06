package tester

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type Tester struct {
	*testing.T
}

// New creates a new tester instance
func New(t *testing.T) Tester {
	return Tester{t}
}

// Assert returns a testify/assert instance
func (t *Tester) Assert() *assert.Assertions {
	return assert.New(t)
}

func (t *Tester) ErrorContains(err error, contains string) {
	if err == nil {
		t.Assert().Error(err)
		return
	}
	t.Assert().Contains(err.Error(), contains)
}

func (t *Tester) PanicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
