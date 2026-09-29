//go:build acceptance && linux

package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// Run the bounded-memory checks with:
//
//	go test -tags acceptance ./pkg/beacon/api -run Acceptance -count=1
//
// The generated transport produces payloads without allocating. Reads use an
// explicit loop so io.Copy fast paths cannot bypass the path under test.
// Workers rendezvous before and midway through reads for deterministic samples.
// GOGC is fixed and GOMEMLIMIT is disabled for the duration of each test.
const (
	beaconStatePayloadBytes int64  = 73_216_552
	streamConcurrency              = 50
	copyBufferSize                 = 32 << 10
	peakHeapBudget          uint64 = 64 << 20
	peakRSSBudget           uint64 = 64 << 20

	comparisonConcurrency         = 8
	comparisonPayloadBytes int64  = 8 << 20
	comparisonStreamBudget uint64 = 16 << 20

	acceptanceGCPercent = 100
	heapSampleInterval  = 5 * time.Millisecond
	acceptanceTimeout   = 10 * time.Minute

	rawContentType  = "application/octet-stream"
	beaconStateID   = "head"
	beaconStatePath = "/eth/v2/debug/beacon/states/" + beaconStateID
)

// TestAcceptanceConcurrentBeaconStateStreamsAreBoundedMemory streams 50
// concurrent 73,216,552-byte beacon states, records peak heap and RSS, and
// asserts that both stay bounded and every byte arrives intact.
func TestAcceptanceConcurrentBeaconStateStreamsAreBoundedMemory(t *testing.T) {
	pinGCSettings(t)

	ctx, cancel := context.WithTimeout(context.Background(), acceptanceTimeout)
	defer cancel()

	client, observer, transport := newAcceptanceClient(beaconStatePayloadBytes)
	report := measureConcurrentStreams(ctx, t, client, observer, streamConcurrency, beaconStatePayloadBytes)
	report.log(t, "50 concurrent streamed beacon states")

	if got, want := transport.requests.Load(), int64(streamConcurrency); got != want {
		t.Errorf("transport served %d requests, want %d", got, want)
	}

	if report.peakDelta() > peakHeapBudget {
		t.Errorf(
			"peak heap delta %s exceeds budget %s; a payload-sized buffer is being held somewhere",
			mib(report.peakDelta()), mib(peakHeapBudget),
		)
	}

	if report.peakRSSDelta() > peakRSSBudget {
		t.Errorf(
			"peak RSS delta %s exceeds budget %s; resident memory is not bounded",
			mib(report.peakRSSDelta()), mib(peakRSSBudget),
		)
	}

	// A run that streams 3.4 GiB while allocating less than one beacon state
	// cannot be buffering payloads anywhere along the path.
	if report.totalAlloc > uint64(beaconStatePayloadBytes) {
		t.Errorf(
			"total allocated %s exceeds a single payload %s; the path is allocating per-payload",
			mib(report.totalAlloc), mib(uint64(beaconStatePayloadBytes)),
		)
	}
}

// TestAcceptanceStreamedVersusBufferedPeakHeap contrasts the streaming and
// buffered paths at capped, reliably measurable parameters. The buffered arm's
// floor is a liveness guarantee rather than an estimate: every worker holds its
// []byte at the gate, so the bytes are unambiguously resident when sampled.
func TestAcceptanceStreamedVersusBufferedPeakHeap(t *testing.T) {
	pinGCSettings(t)

	ctx, cancel := context.WithTimeout(context.Background(), acceptanceTimeout)
	defer cancel()

	streamed := measureStreamedArm(ctx, t)
	streamed.log(t, "streamed arm (capped comparison)")

	buffered := measureBufferedArm(ctx, t)
	buffered.log(t, "buffered arm (capped comparison)")

	liveFloor := uint64(comparisonPayloadBytes) * comparisonConcurrency * 9 / 10

	if buffered.peakDelta() < liveFloor {
		t.Errorf(
			"buffered arm peak delta %s is below the %s that %d live payloads require; the arm did not measure what it claims",
			mib(buffered.peakDelta()), mib(liveFloor), comparisonConcurrency,
		)
	}

	if streamed.peakDelta() > comparisonStreamBudget {
		t.Errorf("streamed arm peak delta %s exceeds budget %s", mib(streamed.peakDelta()), mib(comparisonStreamBudget))
	}

	if streamed.peakDelta()*4 > buffered.peakDelta() {
		t.Errorf(
			"streamed peak delta %s is not decisively below buffered peak delta %s",
			mib(streamed.peakDelta()), mib(buffered.peakDelta()),
		)
	}
}

func measureStreamedArm(ctx context.Context, t *testing.T) memReport {
	t.Helper()

	client, observer, _ := newAcceptanceClient(comparisonPayloadBytes)

	return measureConcurrentStreams(ctx, t, client, observer, comparisonConcurrency, comparisonPayloadBytes)
}

func measureConcurrentStreams(
	ctx context.Context,
	t *testing.T,
	client ConsensusClient,
	observer *countingObserver,
	concurrency int,
	payloadBytes int64,
) memReport {
	t.Helper()

	openGate := newGate(concurrency)
	midGate := newGate(concurrency)
	results := make([]streamResult, concurrency)

	var wg sync.WaitGroup

	baseline := settleAndReadMemStats()
	baselineRSS := mustReadProcessRSS(t)
	sampler := startMemorySampler(heapSampleInterval)
	started := time.Now()

	for i := range concurrency {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			results[idx] = streamOneState(ctx, client, payloadBytes, openGate, midGate)
		}(i)
	}

	if err := openGate.awaitReached(ctx); err != nil {
		t.Fatalf("waiting for streams to open: %v", err)
	}

	atOpen := sampler.sample()

	openGate.release()

	if err := midGate.awaitReached(ctx); err != nil {
		t.Fatalf("waiting for streams to reach mid-body: %v", err)
	}

	atMidpoint := sampler.sample()

	midGate.release()

	wg.Wait()

	elapsed := time.Since(started)
	peak, err := sampler.stopAndWait()
	if err != nil {
		t.Fatalf("sample process memory: %v", err)
	}
	final := readMemStats()

	var totalBytes int64

	for i, result := range results {
		if result.err != nil {
			t.Errorf("stream %d: %v", i, result.err)
		}

		totalBytes += result.copied
	}

	if want := payloadBytes * int64(concurrency); totalBytes != want {
		t.Errorf("copied %d bytes, want %d", totalBytes, want)
	}

	observer.assert(t, int64(concurrency))

	return memReport{
		streams:       concurrency,
		payloadBytes:  payloadBytes,
		totalBytes:    totalBytes,
		duration:      elapsed,
		baselineHeap:  baseline.HeapAlloc,
		peakHeap:      peak.heap,
		atOpen:        atOpen.heap,
		atMidpoint:    atMidpoint.heap,
		baselineRSS:   baselineRSS,
		peakRSS:       peak.rss,
		atOpenRSS:     atOpen.rss,
		atMidpointRSS: atMidpoint.rss,
		totalAlloc:    final.TotalAlloc - baseline.TotalAlloc,
		mallocs:       final.Mallocs - baseline.Mallocs,
		numGC:         final.NumGC - baseline.NumGC,
		pauseTotal:    time.Duration(final.PauseTotalNs - baseline.PauseTotalNs),
	}
}

func measureBufferedArm(ctx context.Context, t *testing.T) memReport {
	t.Helper()

	return measureBuffered(ctx, t, comparisonConcurrency, comparisonPayloadBytes)
}

func measureBuffered(ctx context.Context, t *testing.T, concurrency int, payloadBytes int64) memReport {
	t.Helper()

	client, _, _ := newAcceptanceClient(payloadBytes)

	liveGate := newGate(concurrency)
	results := make([]streamResult, concurrency)
	held := make([][]byte, concurrency)

	var wg sync.WaitGroup

	baseline := settleAndReadMemStats()
	baselineRSS := mustReadProcessRSS(t)
	sampler := startMemorySampler(heapSampleInterval)
	started := time.Now()

	for i := range concurrency {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			results[idx] = bufferOneState(ctx, client, payloadBytes, &held[idx], liveGate)
		}(i)
	}

	if err := liveGate.awaitReached(ctx); err != nil {
		t.Fatalf("buffered arm: waiting for buffers to become live: %v", err)
	}

	atLive := sampler.sample()

	liveGate.release()

	wg.Wait()

	elapsed := time.Since(started)
	peak, err := sampler.stopAndWait()
	if err != nil {
		t.Fatalf("buffered arm: sample process memory: %v", err)
	}
	final := readMemStats()

	runtime.KeepAlive(held)

	var totalBytes int64

	for i, res := range results {
		if res.err != nil {
			t.Errorf("buffered arm: fetch %d: %v", i, res.err)
		}

		totalBytes += res.copied
	}

	if want := payloadBytes * int64(concurrency); totalBytes != want {
		t.Errorf("buffered arm: read %d bytes, want %d", totalBytes, want)
	}

	return memReport{
		streams:       concurrency,
		payloadBytes:  payloadBytes,
		totalBytes:    totalBytes,
		duration:      elapsed,
		baselineHeap:  baseline.HeapAlloc,
		peakHeap:      peak.heap,
		atOpen:        atLive.heap,
		atMidpoint:    atLive.heap,
		baselineRSS:   baselineRSS,
		peakRSS:       peak.rss,
		atOpenRSS:     atLive.rss,
		atMidpointRSS: atLive.rss,
		totalAlloc:    final.TotalAlloc - baseline.TotalAlloc,
		mallocs:       final.Mallocs - baseline.Mallocs,
		numGC:         final.NumGC - baseline.NumGC,
		pauseTotal:    time.Duration(final.PauseTotalNs - baseline.PauseTotalNs),
	}
}

type streamResult struct {
	copied int64
	err    error
}

// streamOneState opens a beacon state, rendezvouses at openGate before reading
// a byte, rendezvouses again at midGate half way through the body, and verifies
// every byte against the generator as it goes.
func streamOneState(
	ctx context.Context,
	client ConsensusClient,
	size int64,
	openGate, midGate *gate,
) (res streamResult) {
	// Any early return must release both gates, or the peers block forever.
	// Releasing early skews the sample, but only on a path that already fails
	// the test.
	defer openGate.abort()
	defer midGate.abort()

	rsp, err := client.OpenRawDebugBeaconState(ctx, beaconStateID, rawContentType)
	if err != nil {
		res.err = fmt.Errorf("open: %w", err)

		return res
	}

	defer func() {
		if cerr := rsp.Close(); cerr != nil && res.err == nil {
			res.err = fmt.Errorf("close: %w", cerr)
		}
	}()

	if rsp.ContentType != rawContentType {
		res.err = fmt.Errorf("content type %q, want %q", rsp.ContentType, rawContentType)

		return res
	}

	if rsp.ContentLength != size {
		res.err = fmt.Errorf("content length %d, want %d", rsp.ContentLength, size)

		return res
	}

	if err := openGate.arrive(ctx); err != nil {
		res.err = fmt.Errorf("open gate: %w", err)

		return res
	}

	var (
		buf      = make([]byte, copyBufferSize)
		sink     = newVerifyingSink()
		midpoint = size / 2
		atMid    bool
	)

	for {
		// Explicit Read loop: io.Copy could take a WriterTo or ReaderFrom fast
		// path and stop measuring the path under test.
		n, readErr := rsp.Read(buf)
		if n > 0 {
			if _, werr := sink.Write(buf[:n]); werr != nil {
				res.err = werr
				res.copied = sink.total

				return res
			}

			if !atMid && sink.total >= midpoint {
				atMid = true

				if err := midGate.arrive(ctx); err != nil {
					res.err = fmt.Errorf("mid gate: %w", err)
					res.copied = sink.total

					return res
				}
			}
		}

		if errors.Is(readErr, io.EOF) {
			break
		}

		if readErr != nil {
			res.err = fmt.Errorf("read at offset %d: %w", sink.total, readErr)
			res.copied = sink.total

			return res
		}
	}

	if !atMid {
		res.err = fmt.Errorf("stream ended at %d bytes without reaching the midpoint %d", sink.total, midpoint)

		return res
	}

	if sink.total != size {
		res.err = fmt.Errorf("streamed %d bytes, want %d", sink.total, size)
	}

	res.copied = sink.total

	return res
}

// bufferOneState fetches through the []byte path and holds the result live at
// the gate, so the comparison arm samples a heap that provably contains every
// payload at once.
func bufferOneState(
	ctx context.Context,
	client ConsensusClient,
	size int64,
	hold *[]byte,
	liveGate *gate,
) (res streamResult) {
	defer liveGate.abort()

	data, err := client.RawDebugBeaconState(ctx, beaconStateID, rawContentType)
	if err != nil {
		res.err = fmt.Errorf("fetch: %w", err)

		return res
	}

	*hold = data
	res.copied = int64(len(data))

	if int64(len(data)) != size {
		res.err = fmt.Errorf("buffered %d bytes, want %d", len(data), size)

		return res
	}

	// Verify in fixed chunks so the verifier's own scratch stays at
	// copyBufferSize; handing it the whole payload would grow a second
	// payload-sized buffer and inflate the very number this arm reports.
	sink := newVerifyingSink()

	for off := 0; off < len(data); off += copyBufferSize {
		if _, err := sink.Write(data[off:min(off+copyBufferSize, len(data))]); err != nil {
			res.err = err

			return res
		}
	}

	if err := liveGate.arrive(ctx); err != nil {
		res.err = fmt.Errorf("live gate: %w", err)
	}

	runtime.KeepAlive(data)

	return res
}

func newAcceptanceClient(size int64) (ConsensusClient, *countingObserver, *generatedTransport) {
	transport := &generatedTransport{size: size}
	observer := &countingObserver{}

	log := logrus.New()
	log.SetOutput(io.Discard)
	log.SetLevel(logrus.ErrorLevel)

	// Timeout is left at zero deliberately: a non-zero Client.Timeout wraps the
	// body in a cancellation shim, and this measurement should see the
	// library's own plumbing and nothing else.
	rawClient := &http.Client{Transport: transport}

	client := NewConsensusClient(log, "http://acceptance.invalid", rawClient, rawClient, nil, observer)

	return client, observer, transport
}

// generatedTransport answers every beacon state request with a synthetic body
// of the configured size. No server, no socket, no buffer.
type generatedTransport struct {
	size     int64
	requests atomic.Int64
}

func (t *generatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Path != beaconStatePath {
		return nil, fmt.Errorf("unexpected request path %q", req.URL.Path)
	}

	if got := req.Header.Get("Accept"); got != rawContentType {
		return nil, fmt.Errorf("unexpected Accept header %q", got)
	}

	t.requests.Add(1)

	header := make(http.Header, 2)
	header.Set("Content-Type", rawContentType)
	header.Set("Eth-Consensus-Version", "gloas")

	return &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Proto:         "HTTP/2.0",
		ProtoMajor:    2,
		ProtoMinor:    0,
		Header:        header,
		Body:          &generatedBody{remaining: t.size},
		ContentLength: t.size,
		Request:       req,
	}, nil
}

var errBodyClosed = errors.New("read on closed generated body")

// patternBlockSize is the size of the shared, read-only block every generated
// body copies from. One block for the whole process keeps body reads free of
// allocation.
const patternBlockSize = 64 << 10

var patternBlock = newPatternBlock()

func newPatternBlock() []byte {
	b := make([]byte, patternBlockSize)

	x := uint32(0x9E3779B9)
	for i := range b {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		b[i] = byte(x)
	}

	return b
}

// fillFromPattern writes the bytes the generator owes at the given absolute
// offset. It allocates nothing and is a pure function of the offset, so the
// reader and the verifier agree without sharing state.
func fillFromPattern(p []byte, offset int64) {
	for n := 0; n < len(p); {
		start := int((offset + int64(n)) % patternBlockSize)
		n += copy(p[n:], patternBlock[start:])
	}
}

// generatedBody is an allocation-free io.ReadCloser. Each body is read by
// exactly one goroutine, so it needs no synchronisation of its own.
type generatedBody struct {
	offset    int64
	remaining int64
	closed    bool
}

func (b *generatedBody) Read(p []byte) (int, error) {
	if b.closed {
		return 0, errBodyClosed
	}

	if b.remaining == 0 {
		return 0, io.EOF
	}

	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}

	fillFromPattern(p, b.offset)

	b.offset += int64(len(p))
	b.remaining -= int64(len(p))

	return len(p), nil
}

func (b *generatedBody) Close() error {
	b.closed = true

	return nil
}

// verifyingSink checks each chunk against the generator at its absolute offset,
// which catches dropped, duplicated or reordered bytes that a plain length
// check would miss. It implements only Write, so no io.ReaderFrom fast path can
// bypass it.
type verifyingSink struct {
	total   int64
	scratch []byte
}

func newVerifyingSink() *verifyingSink {
	return &verifyingSink{scratch: make([]byte, copyBufferSize)}
}

func (s *verifyingSink) Write(p []byte) (int, error) {
	if len(p) > len(s.scratch) {
		s.scratch = make([]byte, len(p))
	}

	want := s.scratch[:len(p)]
	fillFromPattern(want, s.total)

	if !bytes.Equal(p, want) {
		return 0, fmt.Errorf("payload mismatch in the %d bytes at offset %d", len(p), s.total)
	}

	s.total += int64(len(p))

	return len(p), nil
}

// countingObserver records the RawResponse lifecycle so the run can assert that
// ownership was transferred and returned exactly once per stream.
type countingObserver struct {
	opened atomic.Int64
	closed atomic.Int64
	leaked atomic.Int64
}

func (o *countingObserver) RawResponseOpened() { o.opened.Add(1) }
func (o *countingObserver) RawResponseClosed() { o.closed.Add(1) }
func (o *countingObserver) RawResponseLeaked() { o.leaked.Add(1) }

func (o *countingObserver) assert(t *testing.T, want int64) {
	t.Helper()

	if got := o.opened.Load(); got != want {
		t.Errorf("observer saw %d opens, want %d", got, want)
	}

	if got := o.closed.Load(); got != want {
		t.Errorf("observer saw %d closes, want %d", got, want)
	}

	if got := o.leaked.Load(); got != 0 {
		t.Errorf("observer saw %d leaked bodies, want 0", got)
	}
}

// gate is a single-use rendezvous whose release is driven by the test rather
// than by the last arriving worker, so the heap can be sampled at the instant
// every participant is known to be in position.
type gate struct {
	target int

	mu      sync.Mutex
	arrived int

	reachedOnce sync.Once
	reached     chan struct{}

	proceedOnce sync.Once
	proceed     chan struct{}
}

func newGate(target int) *gate {
	return &gate{
		target:  target,
		reached: make(chan struct{}),
		proceed: make(chan struct{}),
	}
}

// arrive reports the caller as in position and blocks until the test releases
// the gate or ctx is done.
func (g *gate) arrive(ctx context.Context) error {
	g.mu.Lock()
	g.arrived++
	full := g.arrived >= g.target
	g.mu.Unlock()

	if full {
		g.markReached()
	}

	select {
	case <-g.proceed:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// awaitReached blocks until every participant has arrived.
func (g *gate) awaitReached(ctx context.Context) error {
	select {
	case <-g.reached:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *gate) release() {
	g.proceedOnce.Do(func() { close(g.proceed) })
}

// abort unblocks everyone regardless of how many arrived. It exists so a failing
// worker cannot deadlock its peers; the failure is reported through the worker's
// own result.
func (g *gate) abort() {
	g.markReached()
	g.release()
}

func (g *gate) markReached() {
	g.reachedOnce.Do(func() { close(g.reached) })
}

type memoryReading struct {
	heap uint64
	rss  uint64
}

// memorySampler tracks heap-object bytes and resident set size across a run.
type memorySampler struct {
	mu       sync.Mutex
	ms       runtime.MemStats
	peakHeap atomic.Uint64
	peakRSS  atomic.Uint64
	err      error

	stop chan struct{}
	done chan struct{}
}

func startMemorySampler(interval time.Duration) *memorySampler {
	s := &memorySampler{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	s.sample()

	go func() {
		defer close(s.done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.sample()
			case <-s.stop:
				return
			}
		}
	}()

	return s
}

// sample records heap-object bytes and resident set size.
func (s *memorySampler) sample() memoryReading {
	s.mu.Lock()
	runtime.ReadMemStats(&s.ms)
	heap := s.ms.HeapAlloc
	rss, err := readProcessRSS()
	if err != nil && s.err == nil {
		s.err = err
	}
	s.mu.Unlock()

	recordPeak(&s.peakHeap, heap)
	recordPeak(&s.peakRSS, rss)

	return memoryReading{heap: heap, rss: rss}
}

func recordPeak(peak *atomic.Uint64, value uint64) {
	for {
		current := peak.Load()
		if value <= current || peak.CompareAndSwap(current, value) {
			return
		}
	}
}

func (s *memorySampler) stopAndWait() (memoryReading, error) {
	close(s.stop)
	<-s.done

	s.sample()

	s.mu.Lock()
	err := s.err
	s.mu.Unlock()

	return memoryReading{
		heap: s.peakHeap.Load(),
		rss:  s.peakRSS.Load(),
	}, err
}

func mustReadProcessRSS(t *testing.T) uint64 {
	t.Helper()

	rss, err := readProcessRSS()
	if err != nil {
		t.Fatalf("read process RSS: %v", err)
	}

	return rss
}

func readProcessRSS() (uint64, error) {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0, err
	}

	fields := bytes.Fields(data)
	if len(fields) < 2 {
		return 0, fmt.Errorf("unexpected /proc/self/statm value %q", data)
	}

	residentPages, err := strconv.ParseUint(string(fields[1]), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse resident pages: %w", err)
	}

	return residentPages * uint64(os.Getpagesize()), nil
}

// settleAndReadMemStats collects twice before reading, so cleanups queued by
// previous work are done and the baseline is live heap rather than residue.
func settleAndReadMemStats() runtime.MemStats {
	runtime.GC()
	runtime.GC()

	return readMemStats()
}

func readMemStats() runtime.MemStats {
	var ms runtime.MemStats

	runtime.ReadMemStats(&ms)

	return ms
}

// pinGCSettings fixes GOGC and GOMEMLIMIT for the test and restores whatever
// the process had before. GOMEMLIMIT is pinned off: a limit would push the GC
// to hold the heap down and would flatter the measurement.
func pinGCSettings(t *testing.T) {
	t.Helper()

	prevGCPercent := debug.SetGCPercent(acceptanceGCPercent)
	prevMemoryLimit := debug.SetMemoryLimit(-1)

	debug.SetMemoryLimit(math.MaxInt64)

	t.Cleanup(func() {
		debug.SetGCPercent(prevGCPercent)
		debug.SetMemoryLimit(prevMemoryLimit)
	})

	t.Logf("GOGC pinned to %d, GOMEMLIMIT pinned off (was %d bytes), GOMAXPROCS=%d",
		acceptanceGCPercent, prevMemoryLimit, runtime.GOMAXPROCS(0))
}

type memReport struct {
	streams       int
	payloadBytes  int64
	totalBytes    int64
	duration      time.Duration
	baselineHeap  uint64
	peakHeap      uint64
	atOpen        uint64
	atMidpoint    uint64
	baselineRSS   uint64
	peakRSS       uint64
	atOpenRSS     uint64
	atMidpointRSS uint64
	totalAlloc    uint64
	mallocs       uint64
	numGC         uint32
	pauseTotal    time.Duration
}

func (r memReport) peakDelta() uint64 {
	if r.peakHeap < r.baselineHeap {
		return 0
	}

	return r.peakHeap - r.baselineHeap
}

func (r memReport) peakRSSDelta() uint64 {
	if r.peakRSS < r.baselineRSS {
		return 0
	}

	return r.peakRSS - r.baselineRSS
}

func (r memReport) log(t *testing.T, name string) {
	t.Helper()

	throughput := float64(r.totalBytes) / (1 << 20) / r.duration.Seconds()

	t.Logf("%s: %d x %d bytes", name, r.streams, r.payloadBytes)
	t.Logf("  total bytes copied  %d (%s)", r.totalBytes, mib(uint64(r.totalBytes)))
	t.Logf("  duration            %s (%.0f MiB/s)", r.duration.Round(time.Millisecond), throughput)
	t.Logf("  heap baseline       %s", mib(r.baselineHeap))
	t.Logf("  heap all-open       %s", mib(r.atOpen))
	t.Logf("  heap mid-stream     %s", mib(r.atMidpoint))
	t.Logf("  heap peak           %s (delta %s)", mib(r.peakHeap), mib(r.peakDelta()))
	t.Logf("  RSS baseline        %s", mib(r.baselineRSS))
	t.Logf("  RSS all-open        %s", mib(r.atOpenRSS))
	t.Logf("  RSS mid-stream      %s", mib(r.atMidpointRSS))
	t.Logf("  RSS peak            %s (delta %s)", mib(r.peakRSS), mib(r.peakRSSDelta()))
	t.Logf("  total allocated     %s in %d allocations", mib(r.totalAlloc), r.mallocs)
	t.Logf("  GC cycles           %d (%s total pause)", r.numGC, r.pauseTotal.Round(time.Microsecond))
}

func mib(b uint64) string {
	return fmt.Sprintf("%.2f MiB", float64(b)/(1<<20))
}
