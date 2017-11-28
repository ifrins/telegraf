# Telegraf Plugin: NATS

This plugin gathers internal stats of NATS. It must be started with [the ```-m``` flag](https://nats.io/documentation/tutorials/nats-monitoring/).

### Configuration:

```
# Read Nginx Plus' advanced status information
[[inputs.nats]]
  ## An array of NATS HTTP server URIs to gather stats.
  urls = ["http://localhost:4222"]
```

### Measurements & Fields:

- nats
  - connections
  - total_connections
  - memory
  - used_cpu
  - routes
  - remotes
  - in_messages
  - out_messages
  - in_bytes
  - out_bytes
  - slow_consumers
  - subscriptions

### Tags:

- nats
  - server
  - port
