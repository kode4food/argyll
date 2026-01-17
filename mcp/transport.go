package mcp

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
)

type (
	rpcEnvelope struct {
		Method string          `json:"method,omitempty"`
		ID     json.RawMessage `json:"id,omitempty"`
	}

	stdioTransport struct {
		in          io.Reader
		out         *bufio.Writer
		msgs        chan []byte
		onClose     func()
		done        chan struct{}
		closedOK    atomic.Bool
		pending     atomic.Int32
		mu          sync.Mutex
		inputOnce   sync.Once
		inputClosed atomic.Bool
	}
)

func newStdioTransport(in io.Reader, out io.Writer) *stdioTransport {
	t := &stdioTransport{
		in:      in,
		out:     bufio.NewWriter(out),
		msgs:    make(chan []byte, 8),
		done:    make(chan struct{}),
		onClose: func() {},
	}
	go t.readLoop()
	return t
}

func (t *stdioTransport) Messages() <-chan []byte {
	return t.msgs
}

func (t *stdioTransport) Send(msg []byte) error {
	if t.closedOK.Load() {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := t.out.Write(msg); err != nil {
		return err
	}
	if err := t.out.WriteByte('\n'); err != nil {
		return err
	}
	if err := t.out.Flush(); err != nil {
		return err
	}
	t.adjustPending(msg)
	t.closeIfIdle()
	return nil
}

func (t *stdioTransport) Close() {
	if t.closedOK.CompareAndSwap(false, true) {
		t.closeInput()
		t.onClose()
	}
}

func (t *stdioTransport) OnClose(fn func()) {
	t.onClose = fn
}

func (t *stdioTransport) readLoop() {
	dec := json.NewDecoder(t.in)
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			t.closeInput()
			return
		}
		t.adjustPending(raw)
		t.msgs <- raw
	}
}

func (t *stdioTransport) closeInput() {
	t.inputOnce.Do(func() {
		t.inputClosed.Store(true)
		close(t.msgs)
		t.closeIfIdle()
	})
}

func (t *stdioTransport) closeIfIdle() {
	if t.inputClosed.Load() && t.pending.Load() == 0 {
		select {
		case <-t.done:
			return
		default:
			close(t.done)
		}
	}
}

func (t *stdioTransport) adjustPending(raw []byte) {
	if len(raw) == 0 {
		return
	}
	delta := countPendingDelta(raw)
	if delta != 0 {
		t.pending.Add(int32(delta))
	}
}

func countPendingDelta(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	if raw[0] == '[' {
		var batch []rpcEnvelope
		if err := json.Unmarshal(raw, &batch); err != nil {
			return 0
		}
		total := 0
		for _, env := range batch {
			total += pendingDelta(env)
		}
		return total
	}
	var env rpcEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return 0
	}
	return pendingDelta(env)
}

func pendingDelta(env rpcEnvelope) int {
	hasID := len(env.ID) > 0
	if hasID && env.Method != "" {
		return 1
	}
	if hasID && env.Method == "" {
		return -1
	}
	return 0
}
