package download_history

import "sync/atomic"

type counter struct {
	count int32
}

func (c *counter) Increment() {
	atomic.AddInt32(&c.count, 1)
}

func (c *counter) Get() int {
	return int(atomic.LoadInt32(&c.count))
}
