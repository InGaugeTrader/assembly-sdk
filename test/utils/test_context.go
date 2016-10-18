package utils

import (
	"fmt"
	"github.com/jonboulle/clockwork"
	"time"
)

type TestContext struct {
	done  chan struct{}
	clock clockwork.Clock
}

func NewTestContextWithTimeout(clock clockwork.Clock, d time.Duration) *TestContext {
	c := &TestContext{
		make(chan struct{}),
		clock,
	}
	go func() {
		<-c.clock.After(d)
		close(c.done)
	}()
	return c
}

func (c *TestContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}
func (c *TestContext) Done() <-chan struct{} {
	return c.done
}
func (c *TestContext) Err() error {
	return fmt.Errorf("some error")
}
func (c *TestContext) Value(key interface{}) interface{} {
	return nil
}
