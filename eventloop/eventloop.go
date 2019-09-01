package eventloop

import (
	"log"
	"sync"

	"github.com/Allenxuxu/gev/poller"
)

// Socket ...
type Socket interface {
	HandleEvent(fd int, events uint32)
}

// EventLoop 事件循环
type EventLoop struct {
	poll      *poller.Poller
	socketers map[int]Socket

	pendingFunc []func()
	mu          sync.Mutex
}

func New() (*EventLoop, error) {
	p, err := poller.Create()
	if err != nil {
		return nil, err
	}

	return &EventLoop{
		poll:      p,
		socketers: make(map[int]Socket),
	}, nil
}

func (l *EventLoop) DeleteFdInLoop(fd int) {
	delete(l.socketers, fd)
}

func (l *EventLoop) AddSocketAndEnableRead(fd int, s Socket) error {
	var err error
	if err = l.poll.AddRead(fd); err != nil {
		return err
	}

	l.socketers[fd] = s
	return nil
}

func (l *EventLoop) EnableReadWrite(fd int) error {
	return l.poll.EnableReadWrite(fd)
}

func (l *EventLoop) EnableRead(fd int) error {
	return l.poll.EnableRead(fd)
}

func (l *EventLoop) RunLoop() {
	l.poll.Poll(l.handlerEvent)
}

func (l *EventLoop) Stop() error {
	return l.poll.Close()
}

func (l *EventLoop) QueueInLoop(f func()) {
	l.mu.Lock()
	l.pendingFunc = append(l.pendingFunc, f)
	l.mu.Unlock()

	if err := l.poll.Wake(); err != nil {
		log.Println("QueueInLoop Wake loop, ", err)
	}
}

func (l *EventLoop) handlerEvent(fd int, events uint32) {
	if fd != -1 {
		s, ok := l.socketers[fd]
		if ok {
			s.HandleEvent(fd, events)
		} else {
			//TODO
			panic("conn not find")
		}
	}

	l.doPendingFunc()
}

func (l *EventLoop) doPendingFunc() {
	l.mu.Lock()
	pf := l.pendingFunc
	l.pendingFunc = nil
	l.mu.Unlock()

	length := len(pf)
	for i := 0; i < length; i++ {
		pf[i]()
	}
}