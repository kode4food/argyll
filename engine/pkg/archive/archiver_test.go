package archive_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/redis"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/archive"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util"
)

const (
	archiveLabelKey   = "tier"
	archiveLabelValue = "archived"
)

func TestNewArchiverValidation(t *testing.T) {
	cfg := archive.Config{
		MemoryCheckInterval: time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	_, err := archive.NewArchiver(nil, nil, cfg)
	assert.Error(t, err)

	cfg.MemoryCheckInterval = 0
	redisClient := goredis.NewClient(&goredis.Options{
		Addr:            "127.0.0.1:1",
		Protocol:        2,
		DisableIdentity: true,
	})
	defer func() { _ = redisClient.Close() }()

	_, err = archive.NewArchiver(&timebox.Store{}, redisClient, cfg)
	assert.Error(t, err)
}

func TestNewArchiverNoPoll(t *testing.T) {
	cfg := archive.Config{
		MemoryCheckInterval: time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:            "127.0.0.1:1",
		Protocol:        2,
		DisableIdentity: true,
	})
	defer func() { _ = redisClient.Close() }()

	arch, err := archive.NewArchiver(&timebox.Store{}, redisClient, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, arch)
}

func TestSweepDeactivated(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	flowStore := setupStore(t, redisServer.Addr())

	id := api.FlowID("flow-sweep")
	seedDeactivatedFlow(t, flowStore, id)
	assertLabelIndexed(t, flowStore, id)

	cfg := archive.Config{
		FlowStore:           config.NewDefaultConfig().FlowStore,
		MemoryPercent:       99.0,
		MaxAge:              0,
		MemoryCheckInterval: time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:            redisServer.Addr(),
		Protocol:        2,
		DisableIdentity: true,
	})
	defer func() { _ = redisClient.Close() }()

	arch, err := archive.NewArchiver(flowStore, redisClient, cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- arch.Run(ctx)
	}()

	var record *timebox.ArchiveRecord
	ok := assert.Eventually(t, func() bool {
		var err error
		record, err = consumeArchive(t.Context(), flowStore, 5*time.Millisecond)
		assert.NoError(t, err)
		return record != nil
	}, testTimeout, testPollInterval)
	if ok {
		assert.Equal(t, "flow:"+string(id), record.AggregateID.Join(":"))
	}

	cancel()
	assert.NoError(t, <-done)

	entries, err := flowStore.ListAggregatesByStatus(events.FlowStatusCompleted)
	assert.NoError(t, err)
	assert.False(t, containsStatusEntry(entries, events.FlowKey(id)))

	assertLabelNotIndexed(t, flowStore, id)
}

func TestPressureArchives(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	flowStore := setupStore(t, redisServer.Addr())

	id := api.FlowID("flow-pressure")
	seedDeactivatedFlow(t, flowStore, id)
	assertLabelIndexed(t, flowStore, id)

	infoAddr, stop := startInfoServer(t, "used_memory:80\nmaxmemory:100\n")
	defer stop()

	cfg := archive.Config{
		FlowStore:           config.NewDefaultConfig().FlowStore,
		MemoryPercent:       50.0,
		MaxAge:              time.Hour,
		MemoryCheckInterval: time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:            infoAddr,
		Protocol:        2,
		DisableIdentity: true,
	})
	defer func() { _ = redisClient.Close() }()

	arch, err := archive.NewArchiver(flowStore, redisClient, cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- arch.Run(ctx)
	}()

	var record *timebox.ArchiveRecord
	ok := assert.Eventually(t, func() bool {
		var err error
		record, err = consumeArchive(t.Context(), flowStore, 5*time.Millisecond)
		assert.NoError(t, err)
		return record != nil
	}, testTimeout, testPollInterval)
	if ok {
		assert.Equal(t, "flow:"+string(id), record.AggregateID.Join(":"))
	}

	cancel()
	assert.NoError(t, <-done)

	entries, err := flowStore.ListAggregatesByStatus(events.FlowStatusCompleted)
	assert.NoError(t, err)
	assert.False(t, containsStatusEntry(entries, events.FlowKey(id)))

	assertLabelNotIndexed(t, flowStore, id)
}

func TestAgeSweepRecent(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	flowStore := setupStore(t, redisServer.Addr())

	id := api.FlowID("flow-recent")
	seedDeactivatedFlow(t, flowStore, id)

	cfg := archive.Config{
		FlowStore:           config.NewDefaultConfig().FlowStore,
		MemoryPercent:       99.0,
		MaxAge:              time.Hour,
		MemoryCheckInterval: time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:            redisServer.Addr(),
		Protocol:        2,
		DisableIdentity: true,
	})
	defer func() { _ = redisClient.Close() }()

	arch, err := archive.NewArchiver(flowStore, redisClient, cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- arch.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	assert.NoError(t, <-done)

	assertNoArchiveStream(t, redisServer)
	assertLabelIndexed(t, flowStore, id)
}

func TestPressureBelowThreshold(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	flowStore := setupStore(t, redisServer.Addr())

	id := api.FlowID("flow-pressure-skip")
	seedDeactivatedFlow(t, flowStore, id)
	assertLabelIndexed(t, flowStore, id)

	infoAddr, stop := startInfoServer(t, "used_memory:40\nmaxmemory:100\n")
	defer stop()

	cfg := archive.Config{
		FlowStore:           config.NewDefaultConfig().FlowStore,
		MemoryPercent:       50.0,
		MaxAge:              time.Hour,
		MemoryCheckInterval: time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:            infoAddr,
		Protocol:        2,
		DisableIdentity: true,
	})
	defer func() { _ = redisClient.Close() }()

	arch, err := archive.NewArchiver(flowStore, redisClient, cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- arch.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	assert.NoError(t, <-done)

	assertNoArchiveStream(t, redisServer)
	assertLabelIndexed(t, flowStore, id)
}

func TestSweepBadStatus(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	flowStore := setupStore(t, redisServer.Addr())

	id := api.FlowID("flow-invalid-status")
	seedDeactivatedFlow(t, flowStore, id)
	assertLabelIndexed(t, flowStore, id)

	cli := goredis.NewClient(&goredis.Options{
		Addr:            redisServer.Addr(),
		Protocol:        2,
		DisableIdentity: true,
	})
	defer func() { _ = cli.Close() }()

	err = cli.ZAdd(t.Context(),
		"partition:idx:status:"+events.FlowStatusCompleted,
		goredis.Z{
			Score:  float64(time.Now().UnixMilli()),
			Member: "bad:flow-id",
		},
	).Err()
	assert.NoError(t, err)

	cfg := archive.Config{
		FlowStore:           config.NewDefaultConfig().FlowStore,
		MemoryPercent:       99.0,
		MaxAge:              0,
		MemoryCheckInterval: time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	arch, err := archive.NewArchiver(flowStore, cli, cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- arch.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	assert.NoError(t, <-done)

	assertNoArchiveStream(t, redisServer)
	assertLabelIndexed(t, flowStore, id)
}

func consumeArchive(
	ctx context.Context, store *timebox.Store, timeout time.Duration,
) (*timebox.ArchiveRecord, error) {
	var record *timebox.ArchiveRecord
	consumeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := store.ConsumeArchive(consumeCtx,
		func(ctx context.Context, rec *timebox.ArchiveRecord) error {
			record = rec
			return nil
		},
	)
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, nil
	}
	return record, err
}

func assertNoArchiveStream(t *testing.T, redisServer *miniredis.Miniredis) {
	t.Helper()
	assert.False(t, redisServer.Exists("partition:archive"))
}

func containsStatusEntry(
	entries []timebox.StatusEntry, id timebox.AggregateID,
) bool {
	for _, entry := range entries {
		if entry.ID == nil {
			continue
		}
		if entry.ID.Join(":") == id.Join(":") {
			return true
		}
	}
	return false
}

func assertLabelIndexed(
	t *testing.T, store *timebox.Store, flowID api.FlowID,
) {
	t.Helper()

	ids, err := store.ListAggregatesByLabel(archiveLabelKey, archiveLabelValue)
	assert.NoError(t, err)
	assert.True(t, containsAggregateID(ids, events.FlowKey(flowID)))
}

func assertLabelNotIndexed(
	t *testing.T, store *timebox.Store, flowID api.FlowID,
) {
	t.Helper()

	ids, err := store.ListAggregatesByLabel(archiveLabelKey, archiveLabelValue)
	assert.NoError(t, err)
	assert.False(t, containsAggregateID(ids, events.FlowKey(flowID)))
}

func containsAggregateID(
	ids []timebox.AggregateID, want timebox.AggregateID,
) bool {
	for _, id := range ids {
		if id.Join(":") == want.Join(":") {
			return true
		}
	}
	return false
}

func setupStore(t *testing.T, redisAddr string) *timebox.Store {
	flowStore, err := redis.NewStore(
		config.NewDefaultConfig().FlowStore,
		redis.Config{
			Addr:   redisAddr,
			Prefix: "partition",
		},
	)
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, flowStore.Close())
	})

	return flowStore
}

func seedDeactivatedFlow(
	t *testing.T, store *timebox.Store, flowID api.FlowID,
) {
	exec := timebox.NewExecutor(
		store, events.NewFlowState, events.FlowAppliers,
	)
	pl := &api.ExecutionPlan{
		Steps:      api.Steps{},
		Attributes: api.AttributeGraph{},
	}
	_, err := exec.Exec(
		events.FlowKey(flowID),
		func(
			st *api.FlowState,
			ag *timebox.Aggregator[*api.FlowState],
		) error {
			if err := events.Raise(
				ag, api.EventTypeFlowStarted,
				api.FlowStartedEvent{
					FlowID: flowID,
					Plan:   pl,
					Init:   api.Args{},
					Labels: api.Labels{
						archiveLabelKey: archiveLabelValue,
					},
				},
			); err != nil {
				return err
			}
			if err := timebox.Raise(ag,
				timebox.EventType(api.EventTypeFlowCompleted),
				api.FlowCompletedEvent{
					FlowID: flowID,
					Result: api.Args{},
				},
			); err != nil {
				return err
			}
			return events.Raise(
				ag, api.EventTypeFlowDeactivated,
				api.FlowDeactivatedEvent{
					FlowID: flowID,
					Status: api.FlowCompleted,
				},
			)
		},
	)
	assert.NoError(t, err)
}

func startInfoServer(t *testing.T, info string) (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	done := make(chan struct{})
	var mu sync.Mutex
	keys := util.Set[string]{}
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
			go handleInfoConn(conn, info, &mu, keys)
		}
	}()

	stop := func() {
		close(done)
		_ = listener.Close()
	}
	return listener.Addr().String(), stop
}

func handleInfoConn(
	conn net.Conn, info string, mu *sync.Mutex, keys util.Set[string],
) {
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
			_ = writeRespBulk(writer, info)
		case "SETNX":
			if len(cmd) < 3 {
				_ = writeRespError(writer, "wrong number of arguments")
				continue
			}
			mu.Lock()
			ok := keys.Contains(cmd[1])
			if !ok {
				keys.Add(cmd[1])
			}
			mu.Unlock()
			_ = writeRespInt(writer, !ok)
		case "SET":
			if len(cmd) < 3 {
				_ = writeRespError(writer, "wrong number of arguments")
				continue
			}
			key := cmd[1]
			setIfMissing := false
			for _, arg := range cmd[3:] {
				if strings.EqualFold(arg, "NX") {
					setIfMissing = true
				}
			}
			mu.Lock()
			exists := keys.Contains(key)
			if !setIfMissing || !exists {
				keys.Add(key)
				mu.Unlock()
				_ = writeRespBulk(writer, "OK")
				continue
			}
			mu.Unlock()
			_, _ = writer.WriteString("$-1\r\n")
			_ = writer.Flush()
		case "DEL":
			if len(cmd) < 2 {
				_ = writeRespError(writer, "wrong number of arguments")
				continue
			}
			var deleted int
			mu.Lock()
			for _, key := range cmd[1:] {
				if keys.Contains(key) {
					keys.Remove(key)
					deleted++
				}
			}
			mu.Unlock()
			_ = writeRespCount(writer, deleted)
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

func writeRespInt(writer *bufio.Writer, ok bool) error {
	if ok {
		_, err := writer.WriteString(":1\r\n")
		if err != nil {
			return err
		}
		return writer.Flush()
	}
	_, err := writer.WriteString(":0\r\n")
	if err != nil {
		return err
	}
	return writer.Flush()
}

func writeRespCount(writer *bufio.Writer, n int) error {
	if _, err := fmt.Fprintf(writer, ":%d\r\n", n); err != nil {
		return err
	}
	return writer.Flush()
}
