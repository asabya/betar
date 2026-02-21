# Mock LLM for E2E Testing

## Goal

Enable full E2E testing of agent execution flow without calling real Gemini API, by implementing a mock LLM that returns predefined responses.

## Design

### Components

1. **MockLLM** (`internal/agent/mock_llm.go`)
   - Implements `model.LLM` interface
   - Stores map of `input pattern -> response`
   - Returns predefined response or echoes input if no match
   - Thread-safe for concurrent test access

2. **MockRuntime** (`internal/agent/mock_runtime.go`)
   - Implements `Runtime` interface
   - Creates agents using ADK's `llmagent.New()` with `MockLLM`
   - Uses in-memory session service (same as ADKRuntime)

3. **Test Integration** (`internal/e2e/e2e_test.go`)
   - Add `NewManagerWithRuntime()` constructor to `Manager`
   - Update E2E tests to use `MockRuntime` instead of real Gemini
   - Test full flow: register → CRDT → discover → execute → response

### Data Flow

```
Test setup → MockRuntime with predefined responses
    ↓
Seller registers agent → MockRuntime creates agent with MockLLM
    ↓
CRDT propagates listing → Buyer discovers agent
    ↓
Buyer sends execute request via P2P → Seller's MockRuntime runs task
    ↓
MockLLM returns predefined response → Buyer receives output
```

### Implementation Details

**MockLLM.GenerateContent:**
- Takes `LLMRequest` with user message
- Looks up response by message text
- Returns `LLMResponse` with matching text or echo

**MockRuntime:**
- Same structure as `ADKRuntime` but without API key requirement
- `CreateAgent()` builds `llmagent` with `MockLLM`
- `RunTask()` iterates events and extracts final response

**Test Usage:**
```go
responses := map[string]string{
    "What is 2+2?": "4",
    "Hello": "Hi there!",
}
runtime := agent.NewMockRuntime(responses)
manager := agent.NewManagerWithRuntime(runtime, ...)
```

## Files Changed

| File | Action |
|------|--------|
| `internal/agent/mock_llm.go` | New |
| `internal/agent/mock_runtime.go` | New |
| `internal/agent/manager.go` | Add `NewManagerWithRuntime()` |
| `internal/e2e/e2e_test.go` | Add `TestE2E_MockLLMExecution` |
