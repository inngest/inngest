package rueidis

import "sync"

type call struct {
	wg *sync.WaitGroup
	mu sync.Mutex
	cn int
}

func (c *call) Do(fn func() error) error {
	var wg *sync.WaitGroup
	c.mu.Lock()
	c.cn++
	if c.wg != nil {
		wg = c.wg
		c.mu.Unlock()
		wg.Wait()
		return nil
	}
	wg = &sync.WaitGroup{}
	wg.Add(1)
	c.wg = wg
	c.mu.Unlock()

	err := fn()
	c.mu.Lock()
	c.wg = nil
	c.cn = 0
	c.mu.Unlock()
	wg.Done()
	return err
}

func (c *call) suppressing() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cn
}
