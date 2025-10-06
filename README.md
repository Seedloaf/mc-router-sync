## MC Router Sync

This is a lightweight sidecar to ensure mc-router is in sync with your list of running servers.

### Configuration

**Note:** entries with a "\*" are required

```
--mc-router-host  | * mc-router API host (e.g. http://localhost:8000)
--server-list-api | * Server list API endpoint (e.g. http://localhost:3000/api/servers)
--auth-type       | Authentication type for the server list API: apikey, none (default: none)
--log-level       | The lowest level log you would like (default: info)
--sync-interval   | Sync interval in seconds (default: 30)
```

#### Auth

If you select `apikey` auth you need to supply the key via the `API_KEY` environment variable. This key will be sent to the Server list API in the following format: `Authorization: Bearer ${API_KEY}`

### Health

There is a server which exposes a `/health` endpoint on port 8080.

### Acknowledgements

Parts of this service were written using AI (Claude Code) - in particular the tests.
