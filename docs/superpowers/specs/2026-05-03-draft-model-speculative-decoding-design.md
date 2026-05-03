# Draft Model / Speculative Decoding

Add speculative decoding support to Shepherd by allowing users to select a smaller, same-architecture model as a "draft model" when loading a main model. The draft model runs inside the same llama-server process via `--draft` and `--draft-max-n` CLI flags.

## Scope

This covers the full-stack feature: backend types, command-line flag construction, architecture validation, API integration, and frontend UI in the load dialog. No new API routes are needed.

## Architecture

### How llama.cpp Speculative Decoding Works

llama-server supports speculative decoding natively:
- `--draft <path>` — path to a draft model GGUF file (same architecture required)
- `--draft-max-n <int>` — max number of draft tokens per speculation round (default: 16)

The draft model is loaded into the same process and shares the KV cache with the main model. The server uses the small draft model to quickly predict candidate tokens, then verifies them against the large model — accepting matching tokens without re-computation.

### Data Flow

```
User selects draft model in LoadModelDialog
  → Frontend sends { draftModelId, draftMaxTokens, enabled.draftModel: true }
    → POST /api/models/:id/load (existing endpoint)
      → Handler resolves draftModelId → model path via ModelManager.GetModel()
      → Validates draft model architecture matches main model
      → LoadRequest.DraftModelPath + DraftMaxTokens passed to BuildStartConfig()
        → llama-server launched with --draft <path> --draft-max-n <n>
```

## Changes

### 1. Backend: LoadRequest Types

**File:** `internal/service/model/types.go`

Add fields to `LoadRequest`:

```go
type LoadRequest struct {
    // ... existing fields ...

    // Speculative decoding / draft model
    DraftModelID   string `json:"draftModelId"`   // Model ID of the draft model (resolved to path server-side)
    DraftMaxTokens int    `json:"draftMaxTokens"`  // Max draft tokens per speculation round (default: 16)
}
```

`DraftModelID` is the frontend-facing field (model ID string). The handler resolves it to `DraftModelPath` before passing to the backend.

### 2. Backend: Command-Line Flag Construction

**File:** `internal/service/model/backend/llamacpp.go`

In `BuildStartConfig()`, after existing flag construction and before the custom command section:

```go
// Speculative decoding / draft model
if req.DraftModelPath != "" {
    args = append(args, "--draft", req.DraftModelPath)
}
if req.DraftMaxTokens > 0 {
    args = append(args, "--draft-max-n", strconv.Itoa(req.DraftMaxTokens))
}
```

### 3. Backend: Handler — Draft Model Resolution & Validation

**File:** `internal/server/handlers_model.go`

In `HandleLoadModel()`, after parsing the JSON body and before calling `modelMgr.LoadAsync()`:

1. If `req.DraftModelID != ""`:
   - Resolve `DraftModelID` to a model via `s.modelMgr.GetModel(req.DraftModelID)`
   - If model not found, return 400 Bad Request
   - Resolve the model's file path (handle shard files — use the first shard)
   - Read the draft model's GGUF metadata to extract architecture
   - Read the main model's GGUF metadata to extract architecture
   - Compare architectures; if mismatch, return 400 with descriptive error
   - Set `req.DraftModelPath` to the resolved path
2. If `req.DraftModelID == ""`, skip (no draft model)

Architecture comparison uses `internal/infra/gguf` parser's `GetArchitecture()` method.

### 4. Backend: LoadRequest Extension

**File:** `internal/service/model/types.go`

Add internal (non-JSON) field for the resolved path:

```go
DraftModelPath string `json:"-"` // Resolved draft model file path (set by handler)
```

This field is set server-side by the handler after validation. It is never serialized from/to JSON.

### 5. Frontend: Types

**File:** `web/src/types/model.ts`

Add to `LoadModelParams`:

```typescript
export interface LoadModelParams {
  // ... existing fields ...

  // Speculative decoding
  draftModelId?: string;       // Model ID of draft model
  draftMaxTokens?: number;     // Max draft tokens per speculation round

  enabled?: {
    // ... existing enabled flags ...
    draftModel?: boolean;      // Enable/disable draft model selection
    draftMaxTokens?: boolean;  // Enable/disable draft max tokens
  };
}
```

### 6. Frontend: LoadModelDialog UI

**File:** `web/src/features/models/components/LoadModelDialog.tsx`

Add a "Speculative Decoding" section in the **left column** (basic config area), positioned after the llama.cpp version selector and before capabilities.

**Draft Model Selector:**
- `SelectInput` component (matching existing UI patterns)
- Label: "Draft 模型（推测解码）"
- Options: filtered from `useModels()` — only models whose `metadata.architecture` matches the main model's architecture
- Include a "不使用" (none) option as default
- Disabled when `enabled.draftModel` switch is off
- Show model name + size in options for easy identification

**Draft Max Tokens:**
- `NumberInput` with enable/disable `Switch`
- Label: "Draft 最大 token 数"
- Default: 16 (llama.cpp default)
- Range: 1–256
- Placed in the right column "Advanced Params" section under a new "Speculative Decoding" subsection

**Filtering logic:**
- Fetch all models via `useModels()`
- Get main model's architecture from `model.metadata.architecture`
- Filter: `m.metadata.architecture === mainArchitecture && m.id !== modelId` (exclude self)
- If no compatible models found, show a hint: "没有找到同架构的候选模型"

**filterEnabledParams update:**
- Add `'draftModelId'` and `'draftMaxTokens'` to the param keys list

### 7. Config Persistence

Draft model settings are saved/loaded through the existing `useModelLoadConfig` / `useSaveModelLoadConfig` hooks. The `LoadRequest` JSON fields (`draftModelId`, `draftMaxTokens`) flow through the existing persistence mechanism — no changes needed to the storage layer.

## Error Handling

| Scenario | Response |
|----------|----------|
| Draft model ID not found | 400: "Draft model not found: {id}" |
| Draft model architecture mismatch | 400: "Draft model architecture ({draftArch}) does not match main model ({mainArch})" |
| Draft model same as main model | 400: "Draft model cannot be the same as the main model" |
| Draft model file not accessible | 400: "Draft model file not accessible: {path}" |

## Files Changed (Summary)

| File | Change |
|------|--------|
| `internal/service/model/types.go` | Add `DraftModelID`, `DraftMaxTokens`, `DraftModelPath` to LoadRequest |
| `internal/service/model/backend/llamacpp.go` | Add `--draft` and `--draft-max-n` flag construction |
| `internal/server/handlers_model.go` | Add draft model resolution + architecture validation |
| `web/src/types/model.ts` | Add `draftModelId`, `draftMaxTokens`, `enabled.draftModel`, `enabled.draftMaxTokens` |
| `web/src/features/models/components/LoadModelDialog.tsx` | Add draft model selector + max tokens input |

## Out of Scope

- Draft model GPU layers (`-ngld`) — can be added later if needed
- Draft model min-p (`--draft-min-p`) — can be added later if needed
- Independent draft model process management
- Draft model VRAM estimation (the existing VRAM estimator doesn't account for dual-model scenarios)
- Backend support for vLLM speculative decoding (only llama.cpp for now)
