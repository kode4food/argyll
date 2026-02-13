package archiver_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"

	"github.com/kode4food/argyll/archiver"
)

func TestNewArchiverValidation(t *testing.T) {
	cfg := archiver.Config{
		MemoryCheckInterval: time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	_, err := archiver.NewArchiver(nil, nil, nil, cfg)
	assert.Error(t, err)

	cfg.MemoryCheckInterval = 0
	redisClient := redis.NewClient(&redis.Options{
		Addr:            "127.0.0.1:1",
		Protocol:        2,
		DisableIdentity: true,
	})
	defer func() { _ = redisClient.Close() }()

	_, err = archiver.NewArchiver(
		&timebox.Store{}, &timebox.Store{}, redisClient, cfg,
	)
	assert.Error(t, err)
}

func TestArchiverSweepDeactivated(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	engineStore, flowStore := setupStores(t, redisServer.Addr())

	flowID := api.FlowID("flow-sweep")
	seedDeactivatedFlow(t, engineStore, flowID)
	seedFlowEvents(t, flowStore, flowID)

	cfg := archiver.Config{
		EngineStore:         timebox.DefaultStoreConfig(),
		FlowStore:           timebox.DefaultStoreConfig(),
		MemoryPercent:       99.0,
		MaxAge:              0,
		MemoryCheckInterval: time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:            redisServer.Addr(),
		Protocol:        2,
		DisableIdentity: true,
	})
	defer func() { _ = redisClient.Close() }()

	arch, err := archiver.NewArchiver(engineStore, flowStore, redisClient, cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- arch.Run(ctx)
	}()

	var record *timebox.ArchiveRecord
	ok := assert.Eventually(t, func() bool {
		err := flowStore.PollArchive(
			context.Background(), 5*time.Millisecond,
			func(ctx context.Context, rec *timebox.ArchiveRecord) error {
				record = rec
				return nil
			},
		)
		assert.NoError(t, err)
		return record != nil
	}, testTimeout, testPollInterval)
	if ok {
		assert.Equal(t, "flow:"+string(flowID), record.AggregateID.Join(":"))
	}

	cancel()
	assert.NoError(t, <-done)

	state := loadEngineState(t, engineStore)
	assert.Empty(t, state.Deactivated)
	assert.Empty(t, state.Archiving)
}

func TestArchiverPressureArchives(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	engineStore, flowStore := setupStores(t, redisServer.Addr())

	flowID := api.FlowID("flow-pressure")
	seedDeactivatedFlow(t, engineStore, flowID)
	seedFlowEvents(t, flowStore, flowID)

	infoAddr, stop := startInfoServer(t, "used_memory:80\nmaxmemory:100\n")
	defer stop()

	cfg := archiver.Config{
		EngineStore:         timebox.DefaultStoreConfig(),
		FlowStore:           timebox.DefaultStoreConfig(),
		MemoryPercent:       50.0,
		MaxAge:              time.Hour,
		MemoryCheckInterval: time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:            infoAddr,
		Protocol:        2,
		DisableIdentity: true,
	})
	defer func() { _ = redisClient.Close() }()

	arch, err := archiver.NewArchiver(engineStore, flowStore, redisClient, cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- arch.Run(ctx)
	}()

	var record *timebox.ArchiveRecord
	ok := assert.Eventually(t, func() bool {
		err := flowStore.PollArchive(
			context.Background(), 5*time.Millisecond,
			func(ctx context.Context, rec *timebox.ArchiveRecord) error {
				record = rec
				return nil
			},
		)
		assert.NoError(t, err)
		return record != nil
	}, testTimeout, testPollInterval)
	if ok {
		assert.Equal(t, "flow:"+string(flowID), record.AggregateID.Join(":"))
	}

	cancel()
	assert.NoError(t, <-done)

	state := loadEngineState(t, engineStore)
	assert.Empty(t, state.Deactivated)
	assert.Empty(t, state.Archiving)
}

func setupStores(
	t *testing.T, redisAddr string,
) (*timebox.Store, *timebox.Store) {
	tbCfg := timebox.DefaultConfig()
	tbCfg.Workers = false
	tb, err := timebox.NewTimebox(tbCfg)
	assert.NoError(t, err)

	engineCfg := timebox.DefaultStoreConfig()
	engineCfg.Addr = redisAddr
	engineCfg.Prefix = "engine"

	flowCfg := timebox.DefaultStoreConfig()
	flowCfg.Addr = redisAddr
	flowCfg.Prefix = "flow"
	flowCfg.Archiving = true

	engineStore, err := tb.NewStore(engineCfg)
	assert.NoError(t, err)

	flowStore, err := tb.NewStore(flowCfg)
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, flowStore.Close())
		assert.NoError(t, engineStore.Close())
		assert.NoError(t, tb.Close())
	})

	return engineStore, flowStore
}

func seedDeactivatedFlow(
	t *testing.T, store *timebox.Store, flowID api.FlowID,
) {
	exec := timebox.NewExecutor(
		store, events.NewEngineState, events.EngineAppliers,
	)
	cmd := func(
		st *api.EngineState, ag *timebox.Aggregator[*api.EngineState],
	) error {
		return timebox.Raise(
			ag,
			timebox.EventType(api.EventTypeFlowDeactivated),
			api.FlowDeactivatedEvent{FlowID: flowID},
		)
	}
	_, err := exec.Exec(context.Background(), events.EngineKey, cmd)
	assert.NoError(t, err)
}

func seedFlowEvents(t *testing.T, store *timebox.Store, flowID api.FlowID) {
	id := events.FlowKey(flowID)
	ev := &timebox.Event{
		Timestamp:   time.Now(),
		AggregateID: id,
		Type:        timebox.EventType("flow_started"),
		Data:        json.RawMessage(`{}`),
	}
	err := store.AppendEvents(context.Background(), id, 0, []*timebox.Event{ev})
	assert.NoError(t, err)
}

func loadEngineState(t *testing.T, store *timebox.Store) *api.EngineState {
	exec := timebox.NewExecutor(
		store, events.NewEngineState, events.EngineAppliers,
	)
	state, err := exec.Exec(
		context.Background(),
		events.EngineKey,
		func(
			st *api.EngineState, ag *timebox.Aggregator[*api.EngineState],
		) error {
			return nil
		},
	)
	assert.NoError(t, err)
	return state
}

func startInfoServer(t *testing.T, info string) (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			go handleInfoConn(conn, info)
		}
	}()

	stop := func() {
		close(done)
		_ = listener.Close()
	}
	return listener.Addr().String(), stop
}

func handleInfoConn(conn net.Conn, info string) {
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		cmd, err := readRespCommand(reader)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				_ = writeRespError(writer, "read failed")
			}
			return
		}
		if len(cmd) == 0 {
			continue
		}

		switch strings.ToUpper(cmd[0]) {
		case "HELLO":
			_ = writeRespError(writer, "unknown command 'HELLO'")
		case "PING":
			_, _ = writer.WriteString("+PONG\r\n")
			_ = writer.Flush()
		case "INFO":
			if len(cmd) >= 2 && strings.EqualFold(cmd[1], "memory") {
				_ = writeRespBulk(writer, info)
			} else {
				_ = writeRespError(writer, "section not supported")
			}
		default:
			_ = writeRespError(writer, "unknown command")
		}
	}
}

func readRespCommand(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\r\n")
	if len(line) == 0 || line[0] != '*' {
		return nil, errors.New("expected array")
	}

	n, err := strconv.Atoi(line[1:])
	if err != nil || n < 0 {
		return nil, errors.New("invalid array length")
	}

	parts := make([]string, 0, n)
	for range n {
		part, err := readRespBulk(reader)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func readRespBulk(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(line, "\r\n")
	if len(line) == 0 || line[0] != '$' {
		return "", errors.New("expected bulk string")
	}

	n, err := strconv.Atoi(line[1:])
	if err != nil || n < 0 {
		return "", errors.New("invalid bulk length")
	}

	buf := make([]byte, n+2)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return "", err
	}
	if string(buf[n:]) != "\r\n" {
		return "", errors.New("invalid bulk terminator")
	}
	return string(buf[:n]), nil
}

func writeRespBulk(writer *bufio.Writer, value string) error {
	if _, err := fmt.Fprintf(
		writer, "$%d\r\n%s\r\n", len(value), value,
	); err != nil {
		return err
	}
	return writer.Flush()
}

func writeRespError(writer *bufio.Writer, msg string) error {
	if _, err := writer.WriteString("-ERR " + msg + "\r\n"); err != nil {
		return err
	}
	return writer.Flush()
}
