# Manager function flow

## Register a custom stream handler to handle p2p streams

```
┌──────────────────────────┐
│  RegisterStreamHandlers()│
│  Registers the execute   │
│  stream handler on the   │
│  libp2p host             │
└────────────┬─────────────┘
             │  incoming P2P stream
             ▼
┌─────────────────────────┐
│  handleExecuteRequest() │
│  Reads & parses the     │
│  execute message from   │
│  the P2P stream         │
└────────────┬────────────┘
             │  parsed TaskRequest
             ▼
┌─────────────────────────┐
│  httpExecuteAndRespond()│
│  Executes the agent     │
│  task and writes the    │
│  TaskResponse back to   │
│  the stream             │
└─────────────────────────┘
```
