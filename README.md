# Beacon

`beacon` is a Go package that can be used to interact with an Ethereum Beacon Node. It provides functions for interacting with a beacon node, and fetches/caches some data from the beacon node to make it easier to use.

As a general rule, any function that starts with `Fetch` will fetch data from the beacon node and cache it. All other functions will use the cached data.

## Features

- Configurable health checks
- Concrete event callback registration
- Fetching/caching of some data from the beacon (like `genesis` and `spec`)

## Built with

- [attestantio/go-eth2-client](github.com/attestantio/go-eth2-client)
- [ethpandaops/ethwallclock](github.com/ethpandaops/ethwallclock)

## Options

Check out the default options in `options.go`

## Installation

```bash
go get github.com/ethpandaops/beacon
```

## Usage

### Simple example

```go
package main

import (
  "context"
  "fmt"
  "log"

  "github.com/ethpandaops/beacon"
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

  "github.com/ethpandaops/beacon"
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

  "github.com/ethpandaops/beacon"
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
