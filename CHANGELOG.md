# Changelog

All notable changes to Shepherd will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

[中文变更日志](CHANGELOG_zh.md)

## [Unreleased]

### Fixed
- **Process management race condition fix**: Fixed "send on closed channel" panic when stopping processes
  - Issue: When health check failure terminates a process, `readOutput` goroutine may still send to a closed channel
  - Solution: Transferred channel close responsibility from `processOutput` to `readOutput` goroutines
  - Uses `atomic.AddInt32` counter to track completed reader count
  - Last reader closes the channel, avoiding race condition

## [v0.6.0] - 2026-03-06

### Breaking Changes
- **Removed internal/master package**: Completely removed master node manager implementation
  - All functionality migrated to `internal/node` package
  - Uses `NodeAdapter` to unify all role API interfaces
  - Removed 2,400+ lines of code, simplifying architecture
  - Aligns with v0.4.0+ unified Node architecture design

### Added
- **LangChainGo Integration**: Added LangChainGo framework integration support
  - New `internal/langchain/` package with 6 core files
  - `LlamaCPP` adapter implements LangChainGo `llms.LLM` interface
  - `Manager` manages multiple LangChainGo instances
  - `Handler` provides HTTP API routes (`/api/langchain/*`)
  - Supports streaming and non-streaming text generation
  - Full unit test coverage (`llama_test.go`, `example_test.go`)
  - Integrated into main startup flow (available in all modes)
- **Utility package**: New `internal/utils/` package providing unified error handling interfaces
  - `CloseQuietly()` - Safely close io.Closer, ignoring errors
  - `RemoveQuietly()` - Safely delete files, no error if not exists
  - `RenameQuietly()` - Safely rename files, log warning on failure
  - `KillQuietly()` - Safely terminate processes, no error if already exited
  - `UnmarshalQuietly()` - Safe JSON parsing with default values on failure
  - `WriteStringQuietly()` - Safely write strings
  - `SetReadDeadlineQuietly()` / `SetWriteDeadlineQuietly()` - Network timeout settings
  - All functions have complete documentation and usage examples
  - Avoids circular dependencies (does not import other internal packages)
- **Process management scripts**: New `scripts/linux/stop_all.sh`
  - Smart detection of Shepherd-related processes (backend + frontend)
  - Graceful shutdown mechanism (SIGTERM + timeout + SIGKILL)
  - Three modes supported:
    - Default: stop all related processes
    - `--dry-run`: preview processes to be stopped (no actual action)
    - `--force`: force stop port occupants
  - Multi-layer detection strategy (pattern match → directory detection → path match)
  - Colored output and detailed logging
  - Cleanup of PID and log files
- **Startup script enhancement**: `scripts/linux/run.sh` integrates process cleanup
  - Automatically calls `stop_all.sh --force` before startup
  - Cleans residual processes, avoids port conflicts
  - Silent execution, failures don't affect startup

### Changed
- **Documentation reorganization**: `doc/` directory migrated to `docs/`
  - More standard directory naming (plural form)
  - New LangChainGo integration documentation:
    - `docs/langchain-integration-summary.md` - Integration overview
    - `docs/langchain-integration-complete.md` - Complete guide
    - `docs/api-vs-langchain-comparison.md` - API comparison
- **Code quality improvements**: Unified use of utility functions for consistency
  - Resource cleanup: `defer rows.Close()` → `defer utils.CloseQuietly(rows)`
  - File operations: `os.Rename()` → `utils.RenameQuietly()`
  - JSON parsing: `json.Unmarshal()` → `utils.UnmarshalQuietly()` (with defaults)
  - Process management: Added `//errcheck:ignore` annotation markers
  - Refactoring across 50+ files
- **Dependency management updates** (go.mod):
  - Added `github.com/tmc/langchaingo v0.1.14`
  - `github.com/ROCm/amdsmi` changed from indirect to direct dependency
  - Updated indirect dependency versions (minor version bumps for multiple packages)

### Removed
- **internal/master package** (2,400+ lines):
  - `doc.go` - Package documentation
  - `handler.go` - Master HTTP handler
  - `node_manager.go` - Node manager (674 lines)
  - `node_manager_test.go` - Node manager tests (482 lines)
  - `scheduler.go` - Scheduler (540 lines)
  - `scheduler_test.go` - Scheduler tests
  - All functionality replaced by `internal/node` + `api.NodeAdapter`
- **doc/ directory** (2,800+ lines):
  - All documentation migrated to `docs/` directory
  - Same directory structure and file names maintained
- **Deprecated API functions**:
  - `server.GetWebSocketManager()` - WebSocket Manager no longer publicly exposed

### Fixed
- **Error handling improvements**: Use `UnmarshalQuietly` to avoid crashes from JSON parse failures
  - Database metadata parsing: added default value logic
  - Session/message metadata parsing: use empty map on failure
  - Model tags/capabilities parsing: use empty slice/empty object on failure
- **Resource cleanup improvements**: Ensure all resources are properly released
  - Database connections, file handles, HTTP servers
  - Graceful shutdown even in error conditions
- **Test code formatting**: Unified test code alignment and indentation

### Migration Guide

If you used the removed `internal/master` package:

1. **Backend code migration**:
   ```go
   // Old code
   import "github.com/simonxluo/Shepherd/internal/master"
   mgr := master.NewNodeManager(...)

   // New code
   import "github.com/simonxluo/Shepherd/internal/node"
   node := node.NewNode(...) // Use unified Node
   ```

2. **API route migration**:
   ```bash
   # Old route (deprecated since v0.4.0+)
   GET /api/master/nodes

   # New route (unified API)
   GET /api/nodes
   ```

3. **Frontend code migration**:
   ```typescript
   // Old code (deprecated since v0.2.0)
   import type { Client } from '@/types/cluster';

   // New code (recommended)
   import type { UnifiedNode } from '@/types/node';
   ```

### Technical Details
- **Architecture simplification**: Removed master-specific code, all roles share one Node implementation
- **Dependency injection**: LangChainGo Handler injected via `RegisterLangChainHandler()`
- **Test strategy**: LangChainGo package uses table-driven tests + example tests
- **Backward compatibility**: v0.4.0+ API routes remain compatible
- **Code statistics**: Net deletion of 5,016 lines (-5,449 +433), improved maintainability

---

## [v0.5.1] - 2026-03-04

### Added
- **One-click startup**: `run.sh` script adds `--web` parameter
  - Starts both frontend and backend dev servers simultaneously
  - Automatically handles port conflicts (detects and stops occupying processes)
  - Graceful exit: `Ctrl+C` stops both frontend and backend
  - Frontend logs output to `/tmp/shepherd-web-dev.log`
  - Frontend process PID saved at `/tmp/shepherd-web-dev.pid`
  - Usage: `./scripts/linux/run.sh --web -b`

### Changed
- **Capability detection optimization**: Refactored `internal/model/capability.go`
  - Extracted 30+ keyword constants, eliminated hardcoding
  - Used `strings.Builder` for string concatenation, reducing 60% memory allocations
  - Unified `ApplyConstraints()` method, eliminating duplicate mutual-exclusion logic
  - Added null checks to avoid unnecessary operations on empty `chat_template`
  - Improved code maintainability and performance

### Performance
- Capability detection memory allocations reduced by 60% (using `strings.Builder`)
- Avoided string conversion on empty `chat_template`

### Code Quality
- Fixed high-priority issues via Simplify Review
- Eliminated duplicate mutual-exclusion logic code
- Extracted keyword constants, improved maintainability

---

## [v0.5.0] - 2026-03-04

### Added
- **Automatic model capability detection**: Auto-detect model capabilities based on model metadata
  - New `internal/model/capability.go` package implementing core detection logic
  - Supports detecting four capabilities: Thinking, Tools, Rerank, Embedding
  - Based on GGUF metadata: model name, architecture, and `chat_template` keyword matching
  - Detection rules:
    - **Thinking**: Matches `deepseek-r1`, `qwq`, `enable_thinking`, `reasoning`, etc.
    - **Tools**: Matches `tool_call`, `function`, `tools`, `mcp`, etc.
    - **Rerank**: Matches `rerank`, `cross-encoder`, `ranker`, etc.
    - **Embedding**: Matches `embedding`, `e5`, `bge`, `jina`, `nomic`, etc.
  - Mutual exclusion rules: Rerank/Embedding mutually exclusive with Thinking/Tools (enabling the former auto-disables the latter)
- **GGUF parser enhancement**: Added `tokenizer.chat_template` field parsing
  - `internal/gguf/metadata.go`: Added `ChatTemplate` field
  - `internal/gguf/parser.go`: Parse `tokenizer.chat_template` KV key
- **Capability persistence**: Migrated model capability storage from memory to database
  - `internal/storage/storage.go`: Added `Capabilities` type definition
  - `internal/storage/sqlite.go`: Added `capabilities` column to `model_metadata` table
  - Implemented database migration logic using `PRAGMA table_info` to check column existence
  - JSON serialized storage to SQLite TEXT column
- **Capability validation methods**: `storage.Capabilities` type adds validation methods
  - `Validate()`: Validate capability configuration (check mutual exclusion rules)
  - `ApplyConstraints()`: Automatically apply mutual exclusion constraints
- **Unit tests**: Added 8 capability detection test cases
  - `TestDetectCapabilities_DeepSeekR1`: DeepSeek-R1 thinking model
  - `TestDetectCapabilities_BGE_M3`: BGE-M3 embedding model
  - `TestDetectCapabilities_BGEReranker`: BGE Reranker re-ranking model
  - `TestDetectCapabilities_GPT4o`: GPT-4o tool-calling model
  - `TestDetectCapabilities_QWQ`: QWQ thinking model
  - `TestDetectCapabilities_E5Embedding`: E5 embedding model
  - `TestDetectCapabilities_NilMetadata`: Nil metadata handling
  - `TestDetectCapabilities_EmptyMetadata`: Empty metadata handling

### Changed
- **Model scan flow**: Integrated capability detection into `loadModel()` function
  - Auto-detect and save capabilities to database during scanning
  - `internal/model/manager.go`: Call `DetectCapabilities()` in `loadModel()`
- **API layer refactoring**: Removed in-memory storage, switched to database
  - `internal/server/server.go`: Removed `capabilities` map and `capabilitiesMu` mutex
  - Deleted local `ModelCapabilities` type definition, unified to `storage.Capabilities`
  - `handleLoadModel()`: Uses `Validate()` and `ApplyConstraints()` methods
  - `handleSetModelCapabilities()`: Uses new validation methods
  - `handleGetModelCapabilities()`: Reads capabilities from database
- **Error handling improvements**: Added warning logs for database save failures
  - `internal/server/server.go`: Log warning when capability save fails
  - `internal/model/manager.go`: Log warning when capability save fails

### Removed
- `docs/llama-server-params-analysis.md`: llama-server parameter analysis doc (outdated)
- `docs/llama-server-params-summary.md`: llama-server parameter summary doc (outdated)
- `web/scripts/check-model-detail.cjs`: Legacy model detail check script
- `web/scripts/check-model-detail.ts`: Legacy model detail check script
- `web/scripts/migrate-text-colors.cjs`: Text color migration script

### Fixed
- **Code quality**: Fixed multiple issues via Simplify Review
  - Extracted duplicate validation logic to `Capabilities.Validate()` method
  - Added error logging (previously used `_` to ignore errors)
  - Unified error message format

### Technical Details
- **Capability detection algorithm**: Keyword matching (case-insensitive)
  - Combined input: `model.Name + " " + model.Architecture` (lowercase)
  - Chat Template detection: Match specific tokens in Jinja templates
- **Database migration**: Uses `ALTER TABLE ADD COLUMN` to dynamically add columns
- **JSON storage**: Uses `encoding/json` to serialize `Capabilities` struct
- **Test coverage**: All 8 test cases pass

---

## [v0.4.0] - 2026-03-01

### BREAKING CHANGES
- **Removed standalone role**: Removed standalone single-machine mode
  - Unified to use `node.role` field as the sole role configuration source
  - Removed `Config.Mode` field, eliminating dual configuration
  - Only three roles remain: `master`, `client`, `hybrid`
- **Default role change**: System default role changed from `standalone` to `hybrid`
  - New users directly use the recommended hybrid mode
  - Hybrid mode enables both Master and Client functionality
- **CLI parameter changes**: Removed `--mode` parameter
  - Node role determined by `node.role` field in config file
  - Run scripts no longer support positional arguments for mode
- **API response changes**:
  - `/config` endpoint returns `role` field instead of `mode`
  - `/api/server/status` endpoint returns `role` field instead of `mode`

### Changed
- **Configuration simplification**:
  - Removed `ConfigFileNames` mapping (mode → config file)
  - `NewManager()` no longer accepts `mode` parameter
  - `NewManagerWithPath()` no longer accepts `mode` parameter
- **Logging system**:
  - Log file naming uses `role` instead of `serverMode`
  - Log file format: `shepherd-{role}-{date}.log`
- **Frontend types**:
  - `NodeRole` type removes `standalone`
  - `ServerModeConfig.mode` field removed
- **Script updates**:
  - `scripts/linux/run.sh` removes mode parameter logic
  - Config files auto-copy from `config/example/` to `config/node/`

### Removed
- `internal/node/types.go`: `NodeRoleStandalone` constant
- `cmd/shepherd/main.go`:
  - `--mode` CLI parameter
  - Mode mapping logic in `determineRole()` function
  - `initStandaloneNode()` function
  - Standalone mode initialization branch
- `internal/config/config.go`:
  - `Config.Mode` field
  - `ConfigFileNames` variable
  - Mode validation logic
- `internal/server/server.go`: `Config.Mode` field
- `internal/logger/logger.go`: `serverMode` field

### Migration Guide
If you are using an older version configuration:

1. **Update config file**:
   ```yaml
   # Remove mode field
   # mode: standalone  # Delete this line

   # Update node.role
   node:
     role: hybrid  # standalone → hybrid (recommended)
   ```

2. **Update startup command**:
   ```bash
   # Old command
   ./build/shepherd standalone
   ./build/shepherd --mode hybrid

   # New command (uses node.role in config file)
   ./build/shepherd
   ./build/shepherd --config config/node/server.config.yaml
   ```

3. **Update frontend code**:
   - `serverConfig.mode` → `serverConfig.role`
   - Remove `'standalone'` type references
   - Use `'hybrid'` instead of `'standalone'`

### Technical Details
- **Config migration**: `syncLegacyConfig()` automatically migrates `mode: standalone` to `node.role: hybrid`
- **Backward compatibility**: Old `master`, `client` config files still work
- **Log compatibility**: Old log files still recognizable, new logs use new naming format

---

## [v0.3.2] - 2026-02-27

### Added
- **Model load config persistence**: Support saving and loading model configurations
  - New `ModelLoadConfig` storage model using `(nodeID, modelID)` as composite primary key
  - New API endpoints:
    - `GET /api/models/:id/load-config` - Get model load config
    - `PUT /api/models/:id/load-config` - Save model load config
    - `DELETE /api/models/:id/load-config` - Delete model load config
  - SQLite storage uses UPSERT pattern, auto-updates existing configs
  - Memory storage uses composite key `"nodeID:modelID"` for config storage
  - NodeAdapter adds `GetNodeID()` method to get node ID
- **Frontend Hooks**: New model config management hooks
  - `useModelLoadConfig()` - Query model config
  - `useSaveModelLoadConfig()` - Save model config
  - `useDeleteModelLoadConfig()` - Delete model config
- **LoadModelDialog enhancements**:
  - Auto-loads last saved config
  - Auto-saves config when loading a model
  - New "Reset" button to clear saved config and restore defaults

### Changed
- **Default storage type**: Example config `server.config.yaml` storage type changed from `memory` to `sqlite`
  - Database path: `./data/shepherd.db`

### Fixed
- **Path config error messages**:
  - PathEditDialog now shows friendly error messages
  - Parses backend error messages, distinguishing "path not found", "not a directory", etc.
  - Dialog stays open on save failure, allowing user to modify and retry
- **Path validation improvements**:
  - Calls backend API for real validation (llama.cpp paths)
  - Validation states: null (unvalidated) / true (valid) / false (invalid)
  - 500ms debounce to avoid frequent requests
  - Shows "Validating path..." loading state
- **Version display update**: Sidebar version updated from v0.1.2 to v0.4.1
- **Log viewer fix**:
  - Fixed log content overlap issue
  - Uses dynamic row height calculation based on message length
  - Updated to react-window v2 API

---

## [v0.3.1] - 2026-02-26

### Added
- **Extended model load parameters**: Added 11 new llama.cpp server parameter support
  - **Sampling parameters**: `reranking`, `minP`, `presencePenalty`, `frequencyPenalty`
  - **Template and processing**: `disableJinja` (--no-jinja), `chatTemplate` (--chat-template), `contextShift` (--context-shift)
  - **KV cache config**: `kvCacheTypeK` (--kv-cache-type-k), `kvCacheTypeV` (--kv-cache-type-v), `kvCacheUnified` (--kv-unified), `kvCacheSize` (--kv-cache-size)
  - Frontend LoadModelDialog fully supports all new parameter configs
  - Backend BuildCommandFromRequest supports all new parameters
- **Logging system enhancement**:
  - All logs automatically include call location (filename:line)
  - Log format adjusted: `[time] [file:line] level message` (text format) / `{"time":"...","caller":"...","level":"...","msg":"..."}` (JSON format)
  - Added detailed logging for model load/unload processes
- **Development experience improvements**:
  - Updated .gitignore to exclude AI editor config files (.claude/, .sisyphus, .cursor/, etc.)
  - CLAUDE.md documentation added to version control

### Fixed
- **Parameter compatibility**: Disabled parameters not supported by llama-server
  - `--logits-all` only works with llama-cli, not llama-server
  - `--dio` requires specific filesystem support, disabled by default
- **Type synchronization**: Updated `toProcessLoadRequest` conversion function to ensure all new fields are correctly passed
- **Test updates**: Updated test cases to reflect disabled parameters

### Technical Details
- **Supported parameter mappings**:
  - `reranking: boolean` → `--reranking`
  - `minP: float64` → `--min-p <value>`
  - `presencePenalty: float64` → `--presence-penalty <value>`
  - `frequencyPenalty: float64` → `--frequency-penalty <value>`
  - `disableJinja: boolean` → `--no-jinja`
  - `chatTemplate: string` → `--chat-template <value>`
  - `contextShift: boolean` → `--context-shift`
  - `kvCacheTypeK: string` → `--kv-cache-type-k <value>`
  - `kvCacheTypeV: string` → `--kv-cache-type-v <value>`
  - `kvCacheUnified: boolean` → `--kv-unified`
  - `kvCacheSize: int` → `--kv-cache-size <value>`

---

## [v0.3.0] - 2026-02-22

### Added
- **HuggingFace SDK integration**: Integrated two HuggingFace Go SDKs
  - `github.com/gomlx/go-huggingface/hub` - Basic Hub operations and file downloads
  - `github.com/bodaay/HuggingFaceModelDownloader` - Advanced downloads (chunked/resumable)
  - Supports Basic download mode and Advanced download mode
  - Added download progress callbacks (speed, ETA, percentage)
  - Supports multiple endpoints (official HuggingFace and HF-Mirror)
  - Frontend adds HuggingFace model search and download UI
- **llama.cpp test functionality**: New `internal/client/tester` package
  - Auto-detects common llama.cpp binary paths
  - Tests binary executability and version info
  - Supports `LLAMACPP_SERVER_PATH` environment variable
  - System info collection (GPU/CPU/ROCm)
  - Frontend adds llama.cpp availability test UI
- **Config report functionality**: New `internal/client/configreport` package
  - Collects llama.cpp path config and availability
  - Collects model paths and model counts
  - Environment info (OS/Kernel/Python/Go versions)
  - Conda config and executor config
  - Frontend adds node config info display
- **NodeAdapter API enhancements**:
  - `POST /api/nodes/:id/test-llamacpp` - Test llama.cpp availability
  - `GET /api/nodes/:id/config` - Get node config info
- **Config validation tests**: Added mode validation tests
  - Validates all valid modes (standalone, hybrid, master, client)
  - Validates invalid mode rejection
- **Path update tests**: Added `TestHandler_UpdateLlamaCppPath` test
  - Tests originalPath matching strategy
  - Tests name-based matching
  - Tests error scenarios
- **Frontend type system**: `web/src/types/node.ts` adds config-related types
  - `NodeConfigInfo` - Node config info
  - `LlamaCppPathInfo` - llama.cpp path info
  - `ModelPathInfo` - Model path info
  - `EnvironmentInfo` - Environment info
  - `CondaConfigInfo` - Conda config
  - `ExecutorConfigInfo` - Executor config

### Changed
- **Path update logic improvement**: Three-level matching strategy
  - Highest priority: `originalPath` exact match
  - Medium priority: Match by `name`
  - Lowest priority: Match by `path` exact
- **Config validation**: Added `standalone` to valid modes list
  - Default mode changed from `hybrid` to `standalone`
  - Supports all four modes: standalone, hybrid, master, client
- **Config deprecation markers**: Added deprecation comments to old Client/Master configs
  - `ClientConfig` marked as deprecated, suggests using `Node.ClientRole`
  - `MasterConfig` marked as deprecated, suggests using `Node.MasterRole`
- **Frontend config management**: Improved config loading and reload logic
  - Supports runtime backend switching
  - Added config validation
- **LoadModelDialog refactoring**: Improved model load dialog
  - Added llama.cpp test button
  - Improved parameter config interface
  - Optimized layout and interaction
- **Settings page enhancement**: Reorganized config items
  - Added environment info display
  - Improved config save logic

### Fixed
- **Path config 500 error**: Fixed config validation rejecting `standalone` mode
- **Path update functionality**: Fixed `UpdateModelPath` and `UpdateLlamaCppPath` matching logic
  - Improved path normalization handling
  - Added better error messages
- **Error message consistency**: Unified error response format
- **Download page**: Fixed download progress display issues
  - Added speed and ETA display
  - Support pause/resume downloads

### Technical Details
- **Download modes**:
  - `DownloadModeBasic`: Uses `go-huggingface/hub` (simple and reliable)
  - `DownloadModeAdvanced`: Uses `bodaay/HuggingFaceModelDownloader` (chunked, resumable)
- **Path matching**: Uses normalized paths for comparison, avoiding symlink issues
- **Test coverage**: Added 5+ test functions covering download, testing, config validation
- **New dependencies**: Two HuggingFace Go SDKs
- **New API commands**:
  - `CommandTypeTestLlamacpp` - Test llama.cpp
  - `CommandTypeGetConfig` - Get node config

---

## [v0.2.0] - 2026-02-22

### Breaking Changes
- **API route unification**: `/api/master/clients/*` and `/api/master/nodes/*` unified to `/api/nodes/*`
  - Old routes marked as deprecated, to be removed in v0.4.0
  - Old routes return `X-API-Deprecation` and `X-API-Sunset` response headers
- **Frontend type refactoring**: `Client` type migrated to `UnifiedNode`
  - `web/src/types/cluster.ts` marked as `@deprecated`
  - Recommend using unified types in `web/src/types/node.ts`

### Added
- **Unified type system**: `internal/types/node.go` adds unified node type definitions
  - `NodeCapabilities` - Node capabilities (GPU/CPU/memory/software support)
  - `NodeResources` - Node resource usage
  - `NodeInfo` - Unified node info (compatible with Master/Client/Node)
  - `HeartbeatMessage` - Unified heartbeat message
  - `Command` and `CommandResult` - Unified command structures
- **Type alias system**: Maintains backward compatibility
  - `internal/node/types.go`: `NodeInfo` → `types.NodeInfo`
  - `internal/cluster/types.go`: `Client` → `types.NodeInfo`
- **Frontend unified types**: `web/src/types/node.ts` adds `UnifiedNode` interface
- **API response helpers**: `internal/handler/response.go` provides unified response format
  - `Success()`, `Error()`, `ValidationError()`, `NotFound()`, etc.
- **Task type definitions**: `web/src/types/task.ts` adds task-related types

### Changed
- **NodeAdapter route refactoring**: Main routes use `/api/nodes/*`
  - `RegisterRoutes()` method refactored
  - Added `registerDeprecatedRoutes()` for old route handling
  - Added `deprecationWarningMiddleware()` deprecation warning middleware
- **Frontend type exports**: `web/src/types/index.ts` reorganized exports
- **Client component updates**: Fixed optional field null handling

### Fixed
- Fixed frontend `ClientCard` component optional field null access issue
- Unified frontend/backend type definitions, eliminated duplication

### Technical Details
- **Backend type unification**: New `internal/types/node.go` as the single source of type definitions
- **Frontend type unification**: `web/src/types/node.ts`'s `UnifiedNode` as recommended type
- **Backward compatibility strategy**: Type aliases keep existing code working without modification
- **API deprecation strategy**: HTTP response headers + log warnings + documentation markers

### Migration Guide

**Backend migration**:
```go
// Old code
import "github.com/simonxluo/Shepherd/internal/node"
nodeInfo := node.NodeInfo{...}

// New code (recommended)
import "github.com/simonxluo/Shepherd/internal/types"
nodeInfo := types.NodeInfo{...}

// Old code still compiles (type alias)
nodeInfo := node.NodeInfo{...} // equivalent to types.NodeInfo
```

**Frontend migration**:
```typescript
// Old code
import type { Client } from '@/types/cluster';

// New code (recommended)
import type { UnifiedNode } from '@/types/node';

// Old code still compiles (type alias)
import type { Client } from '@/types/cluster'; // equivalent to UnifiedNode
```

**API route migration**:
```bash
# Old routes (deprecated)
GET /api/master/clients
GET /api/master/nodes

# New routes (recommended)
GET /api/nodes
```

## [v0.1.4] - 2026-02-22

### Added
- Model benchmark page auto-loads device info from default llama.cpp path
- Device list parsing adds strict prefix validation (ROCm/CUDA/Vulkan/Metal)

### Changed
- Optimized BenchmarkDialog useEffect execution order, ensuring device detection auto-triggers

### Fixed
- Fixed device list parsing incorrectly matching debug info causing duplicate devices
  - llama.cpp --list-devices output contains stderr debug info
  - Before fix: parsed "ggml_cuda_init: found 1 ROCm devices" causing duplicates
  - After fix: only parses the official device list after "Available devices:" marker
- Fixed benchmark dialog not auto-loading devices after opening
  - Adjusted useEffect order, ensuring device detection runs before initialization

### Technical Details
- **Device detection improvement**: `parseDeviceList()` now only starts parsing after finding "Available devices:" marker
- **Device validation**: Added `validDevicePrefix()` method to validate device prefix format
- **Frontend optimization**: BenchmarkDialog useEffect order refactored for auto-loading

## [v0.1.3] - 2026-02-19

### Added
- Config management API (llama.cpp and model paths)
- Download manager full implementation
- Process management API
- Script reorganization (linux/macos/windows)

### Changed
- Web frontend fully independent from backend config
- Path config functionality moved from settings page to independent config

## [v0.1.2] - 2026-02-15

### Added
- Web frontend independent architecture
- Frontend independent config file (web/config.yaml)
- Multi-backend support and runtime switching
- SSE real-time event push

## [v0.1.1] - 2026-02-10

### Added
- Master-Client distributed architecture
- Unified Node model
- Heartbeat and resource monitoring
- Command dispatch and scheduling

## [v0.1.0-alpha] - 2026-02-01

### Added
- Core functionality implementation
- GGUF model scanning and management
- OpenAI/Anthropic/Ollama API compatibility
- Web UI basic features
