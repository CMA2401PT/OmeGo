package task

type statusWaiter struct {
	isInited bool
	waiter   chan int
}

func (w *statusWaiter) wait() {
	if w.isInited {
		return
	} else {
		<-w.waiter
	}
}

func (w *statusWaiter) init() {
	if w.isInited {
		return
	}
	w.isInited = true
	close(w.waiter)
}

func newWaitor() *statusWaiter {
	return &statusWaiter{
		isInited: false,
		waiter:   make(chan int),
	}
}

type HoldedStatus struct {
	isOPWaiter       *statusWaiter
	isOP             bool
	cmdEnabledWaiter *statusWaiter
	cmdEnabled       bool
	cmdFBWaiter      *statusWaiter
	cmdFB            bool
}

func newHolder() *HoldedStatus {
	s := HoldedStatus{
		isOPWaiter:       newWaitor(),
		isOP:             false,
		cmdFBWaiter:      newWaitor(),
		cmdFB:            false,
		cmdEnabledWaiter: newWaitor(),
		cmdEnabled:       false,
	}
	return &s
}

func (s *HoldedStatus) setCmdEnabled(v bool) {
	s.cmdEnabled = v
	s.cmdEnabledWaiter.init()
}

func (s *HoldedStatus) CmdEnabled() bool {
	s.cmdEnabledWaiter.wait()
	return s.cmdEnabled
}

func (s *HoldedStatus) setIsOP(v bool) {
	s.isOP = v
	s.isOPWaiter.init()
}

func (s *HoldedStatus) IsOP() bool {
	s.isOPWaiter.wait()
	return s.isOP
}

func (s *HoldedStatus) setCmdFB(v bool) {
	s.cmdFB = v
	s.cmdFBWaiter.init()
}

func (s *HoldedStatus) CmdFB() bool {
	s.cmdFBWaiter.wait()
	return s.cmdFB
}
