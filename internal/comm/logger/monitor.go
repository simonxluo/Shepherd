package logger

import (
	"io"
	"os"
	"sync"
)

const LogBufferSize = 512 * 1024

type circularBuffer struct {
	data []byte
	head int
	size int
}

func newCircularBuffer(capacity int) *circularBuffer {
	return &circularBuffer{
		data: make([]byte, capacity),
		head: 0,
		size: 0,
	}
}

func (cb *circularBuffer) Write(p []byte) {
	if len(p) == 0 {
		return
	}
	cap := len(cb.data)
	if len(p) >= cap {
		copy(cb.data, p[len(p)-cap:])
		cb.head = 0
		cb.size = cap
		return
	}
	firstPart := cap - cb.head
	if firstPart >= len(p) {
		copy(cb.data[cb.head:], p)
		cb.head = (cb.head + len(p)) % cap
	} else {
		copy(cb.data[cb.head:], p[:firstPart])
		copy(cb.data[:len(p)-firstPart], p[firstPart:])
		cb.head = len(p) - firstPart
	}
	cb.size += len(p)
	if cb.size > cap {
		cb.size = cap
	}
}

func (cb *circularBuffer) GetHistory() []byte {
	if cb.size == 0 {
		return nil
	}
	result := make([]byte, cb.size)
	cap := len(cb.data)
	start := (cb.head - cb.size + cap) % cap
	if start+cb.size <= cap {
		copy(result, cb.data[start:start+cb.size])
	} else {
		firstPart := cap - start
		copy(result[:firstPart], cb.data[start:])
		copy(result[firstPart:], cb.data[:cb.size-firstPart])
	}
	return result
}

type LogDataEvent struct {
	Data []byte
}

type LogMonitor struct {
	bufferMu    sync.RWMutex
	buffer      *circularBuffer
	stdout      io.Writer
	subMu       sync.RWMutex
	subscribers map[chan []byte]struct{}
}

func NewLogMonitor() *LogMonitor {
	return &LogMonitor{
		stdout:      os.Stdout,
		buffer:      nil,
		subscribers: make(map[chan []byte]struct{}),
	}
}

func NewLogMonitorWriter(stdout io.Writer) *LogMonitor {
	return &LogMonitor{
		stdout:      stdout,
		buffer:      nil,
		subscribers: make(map[chan []byte]struct{}),
	}
}

func (m *LogMonitor) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	n, err = m.stdout.Write(p)
	if err != nil {
		return n, err
	}

	m.bufferMu.Lock()
	if m.buffer == nil {
		m.buffer = newCircularBuffer(LogBufferSize)
	}
	m.buffer.Write(p)
	m.bufferMu.Unlock()

	bufCopy := make([]byte, len(p))
	copy(bufCopy, p)

	m.subMu.RLock()
	for ch := range m.subscribers {
		select {
		case ch <- bufCopy:
		default:
		}
	}
	m.subMu.RUnlock()

	return n, nil
}

func (m *LogMonitor) GetHistory() []byte {
	m.bufferMu.RLock()
	defer m.bufferMu.RUnlock()
	if m.buffer == nil {
		return nil
	}
	return m.buffer.GetHistory()
}

func (m *LogMonitor) Clear() {
	m.bufferMu.Lock()
	m.buffer = nil
	m.bufferMu.Unlock()
}

func (m *LogMonitor) Subscribe() chan []byte {
	ch := make(chan []byte, 100)
	m.subMu.Lock()
	m.subscribers[ch] = struct{}{}
	m.subMu.Unlock()
	return ch
}

func (m *LogMonitor) Unsubscribe(ch chan []byte) {
	m.subMu.Lock()
	delete(m.subscribers, ch)
	close(ch)
	m.subMu.Unlock()
}
