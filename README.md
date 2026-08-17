# Beacon

`beacon` is a Go package that can be used to interact with an Ethereum Beacon Node. It provides functions for interacting with a beacon node, and fetches/caches some data from the beacon node to make it easier to use.

As a general rule, any function that starts with `Fetch` will fetch data from the beacon node and cache it. All other functions will use the cached data.

## Features

- Configurable health checks
- Concrete event callback registration
- Fetching/caching of some data from the beacon (like `genesis` and `spec`)

## Built with

- [attestantio/go-eth2-client](github.com/ethpandaops/go-eth2-client)
- [ethpandaops/ethwallclock](github.com/ethpandaops/ethwallclock)

## Options

Check out the default options in `options.go`

## Installation

```bash
go get github.com/ethpandaops/beacon
```

## Usage

### Streaming raw responses

The `OpenRaw*` methods return after the node accepts the request and sends response headers. The caller must close the response and treat any later read error as a failed transfer.

```go
func archiveState(ctx context.Context, node beacon.Node, dst io.Writer) error {
	response, err := node.OpenRawBeaconState(ctx, "finalized", "application/octet-stream")
	if err != nil {
		return err
	}
	defer response.Close()

	mediaType, _, err := mime.ParseMediaType(response.ContentType)
	if err != nil || mediaType != "application/octet-stream" {
		return fmt.Errorf("unexpected content type %q", response.ContentType)
	}

	written, err := io.Copy(dst, response)
	if err != nil {
		return err
	}
	if response.ContentLength >= 0 && written != response.ContentLength {
		return fmt.Errorf("copied %d bytes, expected %d", written, response.ContentLength)
	}

	return nil
}
```

The request context and configured raw API client timeout both govern body reads. The raw client settings apply to both `FetchRaw*` and `OpenRaw*` calls; set `SetAPIClientTimeout(0)` only when every call has a suitable context deadline. By default the raw client permits 32 active response bodies and 32 connections per host; `MaxConcurrentRequests` also caps multiplexed HTTP/2 streams.

Go may transparently decompress gzip responses, reported by `RawResponse.Uncompressed`. Call `Options.SetAPIClientAcceptEncoding("gzip")` when the caller needs the encoded wire bytes; the returned headers then retain `Content-Encoding`. A `ContentLength` of `-1` means the caller cannot verify an expected byte count from that field. Slow readers keep upstream resources open, so applications should bound concurrency and finish downstream writes before committing their output.

Maintainers can run the streaming memory gates with `go test -tags acceptance ./pkg/beacon/api -run Acceptance -count=1`. The full 50-response buffered comparison requires at least 10 GiB of available memory and is intentionally manual:

```bash
go test -tags "acceptance acceptance_full_buffered" ./pkg/beacon/api -run TestAcceptanceFullBufferedBaseline -count=1 -timeout=20m
```

### Simple example

```go
package main

import (
  "context"
  "fmt"
  "log"

  "github.com/ethpandaops/beacon/pkg/beacon"
)

func main() {
  // Create options
  opts := *beacon.DefaultOptions()

  // Create beacon node instance
  beaconNode := beacon.NewNode(e.log, &beacon.Config{
    Addr: "localhost:5052",
    Name: "beacon node",
  }, "eth", opts)

  // Start the beacon node. Start will wait until the beacon node is ready.
  if err := beaconNode.Start(context.Background()); err != nil {
    log.Fatal(err)
  }

  block, err := beaconNode.FetchBlock(context.Background(), "head")
  if err != nil {
    log.Fatal(err)
  }

  fmt.Println(block)
}
```

### Async ready example

```go
package main

import (
  "context"
  "fmt"
  "log"

  "github.com/ethpandaops/beacon/pkg/beacon"
)

func main() {
  // Create options
  opts := *beacon.DefaultOptions()

  // Create beacon node instance
  beaconNode := beacon.NewNode(e.log, &beacon.Config{
    Addr: "localhost:5052",
    Name: "beacon node",
  }, "eth", opts)

  // Register a callback that will be called when the beacon node is ready.
  beaconNode.OnReady(func(ctx context.Context, event *ReadyEvent) error {
    block, err := beaconNode.FetchBlock(context.Background(), "head")
    if err != nil {
      return err
    }

    fmt.Println(block)

    return nil
  })

  // Start the beacon node. StartAsync will start the beacon node in the background.
  if err := beaconNode.StartAsync(context.Background()); err != nil {
    log.Fatal(err)
  }
}
```

### Beacon Events

```go
package main

import (
  "context"
  "fmt"
  "log"

  "github.com/ethpandaops/beacon/pkg/beacon"
)

func main() {
  // Create options
  opts := *beacon.DefaultOptions()

  // Create beacon node instance
  beaconNode := beacon.NewNode(e.log, &beacon.Config{
    Addr: "localhost:5052",
    Name: "beacon node",
  }, "eth", opts)

  // Register a callback that will be called when the beacon node is ready.
  beaconNode.OnBlock(func(ctx context.Context, event *v1.BlockEvent) error {
    fmt.Println(block)

    return nil
  })

  // Start the beacon node. StartAsync will start the beacon node in the background.
  if err := beaconNode.StartAsync(context.Background()); err != nil {
    log.Fatal(err)
  }
}
```
