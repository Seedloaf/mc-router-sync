## MC Router Sync

A lightweight service that automatically synchronizes your Minecraft server list with [mc-router](https://github.com/itzg/mc-router).

MC Router Sync monitors your server inventory and keeps mc-router's routing table up to date. You maintain your server list in whatever format works best for your infrastructure: a file, database, key-value store, or custom API and MC Router Sync handles the synchronization automatically.

## Deployment Options

MC Router Sync can be deployed in two ways:

1. **Sidecar container**: Run as a standalone service that fetches server data from an HTTP API endpoint. Your API must return server information in the format specified below.
   - **Note:** In this configuration you are responsible for providing this HTTP API endpoint.
1. **Embedded in Go applications**: Import MC Router Sync as a library and provide a custom `ServerList` implementation that retrieves server data from your existing infrastructure.

### Configuration

**Note:** entries with a "\*" are required

```
--mc-router-host  | * mc-router API host (e.g. http://localhost:8000)
--server-list-api | * Server list API endpoint (e.g. http://localhost:3000/api/servers)
--auth-type       | Authentication type for the server list API: apikey, none (default: none)
--log-level       | The lowest level log you would like (default: info)
--sync-interval   | Sync interval in seconds (default: 30)
```

### Auth

If you select `apikey` auth you need to supply the key via the `API_KEY` environment variable. This key will be sent to the Server list API in the following format: `Authorization: Bearer ${API_KEY}`

### Health

There is a server which exposes a `/health` endpoint on port 8080.

## Usage Examples

### Example 1: Docker Compose with mc-router

This example shows how to run mc-router-sync as a sidecar container alongside mc-router using Docker Compose:

```yaml
version: "3.8"

services:
  mc-router:
    image: itzg/mc-router:latest
    ports:
      - "25565:25565"
    environment:
      API_BINDING: "0.0.0.0:8000"

  mc-router-sync:
    image: seedloaf/mc-router-discovery:latest
    depends_on:
      - mc-router
    environment:
      # Optional: Set API_KEY if using apikey auth
      # API_KEY: your-api-key-here
    command:
      - "--mc-router-host=http://mc-router:8000"
      - "--server-list-api=http://your-server-list-api:3000/api/servers"
      - "--auth-type=none"
      - "--log-level=info"
      - "--sync-interval=30"
```

Your server list API should return JSON in the following format:

```json
[
  {
    "serverAddress": "lobby.example.com",
    "backend": "localhost:25566"
  },
  {
    "serverAddress": "survival.example.com",
    "backend": "localhost:25567"
  }
]
```

### Example 2: Embedding in a Go Project

You can use the `Reconciler` directly in your Go project with a custom `ServerList` implementation:

```go
package main

import (
    "context"
    "time"

    mcrouterdiscovery "github.com/Seedloaf/mc-router-discovery"
)

type CustomServerList struct {}

func (c *CustomServerList) GetServers() (mcrouterdiscovery.Routes, error) {
    return mcrouterdiscovery.Routes{
        {
            ServerAddress: "lobby.example.com",
            Backend:       "localhost:25566",
        },
        {
            ServerAddress: "survival.example.com",
            Backend:       "localhost:25567",
        },
    }, nil
}

func main() {
    serverList := &CustomServerList{}

    mcRouter := mcrouterdiscovery.NewMcRouterClient("http://localhost:8000")
    reconciler := mcrouterdiscovery.NewReconciler(
        serverList,
        mcRouter,
        30*time.Second,
    )

    ctx := context.Background()
    reconciler.Start(ctx)
}
```

To use this in your project:

```bash
go get github.com/Seedloaf/mc-router-discovery
```

The `ServerList` interface only requires one method:

```go
type ServerList interface {
    GetServers() (Routes, error)
}
```

This allows you to implement server discovery from any source: databases, Kubernetes services, Consul, etcd, or any custom backend.

### Acknowledgements

Parts of this service were written using AI (Claude Code) - in particular the tests.
