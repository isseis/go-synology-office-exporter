# Thread Safety in download_history Package

This document outlines the thread safety guarantees provided by the `download_history` package and highlights important considerations for developers working with it.

## Thread Safety Guarantees

The `DownloadHistory` type provides the following thread safety guarantees:

1. **Concurrent Read Access**: Multiple goroutines can safely call read methods concurrently.
2. **Exclusive Write Access**: Write operations are mutually exclusive with each other and with read operations.
3. **State Consistency**: The internal state remains consistent even when accessed concurrently.
4. **Atomic Counters**: All counter operations are atomic and can be safely accessed concurrently.

## Implementation Details

### Synchronization Primitives

- **Mutex**: A `sync.RWMutex` (`mu`) is used to protect access to shared state.
- **Atomic Operations**: Counters use `sync/atomic` for thread-safe increments and reads.

### Protected State

The following fields are protected by the RWMutex:
- `items` (map of DownloadItem)
- `state` (current state of the history)
- `path` (file path for persistence)

### Thread-Safe Methods

All public methods are safe for concurrent use. They follow these patterns:

- **Read Operations** (use `RLock`/`RUnlock`):
  - `GetItem()`
  - `GetStats()`
  - `GetObsoleteItems()` (requires `stateSaved`)

- **Write Operations** (use `Lock`/`Unlock`):
  - `MarkSkipped()` (requires `stateReady`)
  - `SetDownloaded()` (requires `stateReady`)
  - `Load()`
  - `Save()` (transitions to `stateSaved`)

### State Machine

The `DownloadHistory` follows a strict state machine:
1. `stateNew` → (Load) → `stateReady` → (Save) → `stateSaved`
2. Once in `stateSaved`, only read operations are allowed
3. Most operations require the state to be `stateReady`
4. `GetObsoleteItems()` requires the state to be `stateSaved`

## Developer Guidelines

### What's Safe

1. **Concurrent Reads**: Multiple goroutines can safely call read methods.
   ```go
   // Safe to call from multiple goroutines
   item, exists, err := history.GetItem("some-key")
   stats, err := history.GetStats()

   // GetObsoleteItems requires Save() to be called first
   if err := history.Save(); err == nil {
       obsolete, err := history.GetObsoleteItems()
       // handle obsolete items
   }
   ```

2. **Read-Modify-Write Patterns**: Use the provided atomic methods for counters:
   ```go
   // Thread-safe counter operations
   history.DownloadCount.Increment()
   count := history.DownloadCount.Get()
   ```

### What to Watch Out For

1. **State Dependencies**: Check error returns from methods that verify state:
   ```go
   if err := history.MarkSkipped("key"); err != nil {
       // Handle error (e.g., not ready, not found, etc.)
   }
   ```

2. **No Batch Operations**: Each operation is atomic. For multiple updates, consider:
   - Creating a new method that handles the batch operation under a single lock
   - Or accept that operations might interleave with other goroutines

3. **State Transitions**: Be aware of state requirements for each method. For example, `GetObsoleteItems()` can only be called after `Save()` has been called.

4. **Error Handling**: Most methods can return errors. Always check them, especially for state-related errors like `ErrNotReady`.

### Anti-Patterns to Avoid

1. **Direct Field Access**: Never access struct fields directly:
   ```go
   // UNSAFE - breaks encapsulation and thread safety
   item := history.items["key"]
   ```

2. **Ignoring State**: Don't ignore the state machine:
   ```go
   // BAD - might be called before Load()
   history.MarkSkipped("key")

   // GOOD - check error
   if err := history.MarkSkipped("key"); err != nil {
       // Handle error
   }
   ```

## Performance Considerations

1. **Read Locks**: Read locks are used where possible to allow concurrent reads.
2. **Lock Duration**: Locks are held only for the duration of the critical section.
3. **No Lock Contention**: The implementation minimizes lock contention by using:
   - Fine-grained locking
   - Copy-on-write patterns where appropriate
   - Atomic operations for counters

## Testing

Thread safety is verified through:
1. Race detector (`go test -race`)
2. Concurrent test cases in `*_test.go` files
3. State machine validation

Always run tests with the race detector enabled when making changes to ensure thread safety is maintained.
