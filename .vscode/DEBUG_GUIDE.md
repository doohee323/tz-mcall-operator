# 🔍 Controller Test Debugging Guide

## Step Into Debugging Methods

### 1. Step Into Debugging in VS Code

#### Basic Setup
1. Press **F5** or select "Debug Tests" from `Run and Debug` panel
2. Set breakpoints in `controller_test.go`
3. Use **F11** (Step Into) or **F10** (Step Over)

#### Key Step Into Points

| Test Function | Step Into Target | controller.go Function |
|---------------|------------------|------------------------|
| `TestExecuteCommand` | `executeCommand()` | `executeCommand()` (line 471) |
| `TestTaskWorkerWithExpect` | `NewTaskWorker()` | `NewTaskWorker()` (line 612) |
| `TestTaskWorkerWithExpect` | `worker.Execute()` | `Execute()` (line 623) |
| `TestCheckExpect` | `checkExpect()` | `checkExpect()` (line 999) |
| `TestExecuteHTTPRequest` | `executeHTTPRequest()` | `executeHTTPRequest()` (line 510) |

### 2. Step-by-Step Debugging Guide

#### A. TestExecuteCommand Debugging
```go
// 1. Breakpoint location
output, err := executeCommand(tt.command, tt.timeout)  // ← Set breakpoint here

// 2. Step Into (F11) to controller.go's executeCommand function
// 3. Debug the following functions in order:
//    - strings.Fields(command) - Command parsing
//    - exec.CommandContext() - Command execution
//    - cmd.CombinedOutput() - Result collection
```

#### B. TestTaskWorkerWithExpect Debugging
```go
// 1. Breakpoint locations
worker := NewTaskWorker(tt.input, tt.inputType, tt.nameStr, tt.expect)  // ← Set breakpoint here
worker.Execute(tt.timeout)  // ← Set breakpoint here

// 2. Step Into to debug the following functions:
//    - NewTaskWorker() - TaskWorker struct creation
//    - Execute() - Actual task execution
//    - executeCommand() or executeHTTPRequest() - Type-specific execution
//    - checkExpect() - Result validation
```

### 3. Debugging Commands

#### Terminal Debugging
```bash
# Debug specific test only
make test-specific

# Run all tests with verbose logging
make test-verbose

# Run tests with coverage
make test-coverage
```

#### Using Go Debugger
```bash
# Direct debugging with dlv debugger
dlv test ./controller -- -test.run TestExecuteCommand

# Set breakpoints
(dlv) break controller.executeCommand
(dlv) break controller.NewTaskWorker
(dlv) break controller.(*TaskWorker).Execute

# Execute
(dlv) continue
(dlv) step
(dlv) next
```

### 4. Key Debugging Points

#### Main Functions in controller.go
1. **`executeCommand()`** (lines 471-507)
   - Command parsing and execution
   - Timeout handling
   - Error handling

2. **`executeHTTPRequest()`** (lines 510-553)
   - HTTP request creation
   - Response processing
   - User-Agent setting

3. **`NewTaskWorker()`** (lines 612-620)
   - TaskWorker struct creation
   - Channel initialization

4. **`Execute()`** (lines 623-684)
   - Task type-specific execution branching
   - Expect validation
   - Result generation

5. **`checkExpect()`** (lines 999-1022)
   - Expect string validation
   - OR condition processing (| separator)

### 5. Debugging Tips

#### Variable Inspection
- Check variables in current scope in **Variables panel**
- Monitor specific variables in **Watch panel**
- Check function call stack in **Call Stack**

#### Log Checking
- Check `debugLog()` output in **Debug Console**
- Check detailed logs in **Terminal**

#### Breakpoint Management
- **Conditional breakpoints**: Break only under specific conditions
- **Logpoints**: Output logs without breaking
- **Function breakpoints**: Break when entering specific functions

### 6. Troubleshooting

#### When Step Into Doesn't Work
1. Check if Go extension is up to date
2. Verify `go.mod` file is correct
3. Clear build cache: `go clean -cache`

#### When Debugging is Slow
1. Remove unnecessary breakpoints
2. Run specific tests only with `-test.run`
3. Set timeout with `-test.timeout`

### 7. Example Debugging Scenarios

#### Scenario 1: Command Execution Failure Debugging
1. Set breakpoint on "invalid command" case in `TestExecuteCommand`
2. Step Into to `executeCommand()`
3. Check `exec.CommandContext()` execution
4. Identify error occurrence point

#### Scenario 2: HTTP Request Debugging
1. Set breakpoint on "valid GET request" case in `TestExecuteHTTPRequest`
2. Step Into to `executeHTTPRequest()`
3. Check HTTP request creation process
4. Check response processing process

#### Scenario 3: TaskWorker Execution Debugging
1. Set breakpoint on "cmd with expect match" case in `TestTaskWorkerWithExpect`
2. Step Into to `NewTaskWorker()`
3. Step Into to `Execute()`
4. Check `executeCommand()` call
5. Check `checkExpect()` validation
