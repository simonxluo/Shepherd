# Draft Model / Speculative Decoding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add speculative decoding support by allowing users to select a same-architecture draft model when loading a main model, passing `--draft` and `--draft-max-n` to llama-server.

**Architecture:** Draft model ID is sent from the frontend, resolved to a file path server-side in the HTTP handler, validated for architecture compatibility, then forwarded to the llama.cpp backend's command builder.

**Tech Stack:** Go (Gin handlers, model service), TypeScript/React (dialog UI, types), llama.cpp `--draft`/`--draft-max-n` flags.

---

## File Structure

| File | Change | Responsibility |
|------|--------|----------------|
| `internal/service/model/types.go` | Modify | Add `DraftModelID`, `DraftModelPath`, `DraftMaxTokens` fields to `LoadRequest` |
| `internal/service/model/backend/backend.go` | Modify | Add `DraftModelPath`, `DraftMaxTokens` to `LoadRequest` + `LlamacppLoadParams` |
| `internal/service/model/backend/llamacpp.go` | Modify | Add `--draft` and `--draft-max-n` flag construction |
| `internal/service/model/load.go` | Modify | Map draft fields in `toBackendLoadRequest()` |
| `internal/server/handlers_model.go` | Modify | Resolve draft model ID → path, validate architecture |
| `web/src/types/model.ts` | Modify | Add `draftModelId`, `draftMaxTokens`, enabled flags |
| `web/src/features/models/components/LoadModelDialog.tsx` | Modify | Add draft model selector + max tokens input |

---

### Task 1: Backend — Add draft model fields to types

**Files:**
- Modify: `internal/service/model/types.go:281-285`
- Modify: `internal/service/model/backend/backend.go:76-94`
- Modify: `internal/service/model/backend/backend.go:117-173`

- [ ] **Step 1: Add draft fields to `LoadRequest` in `internal/service/model/types.go`**

Insert after the `RopeFreqScale` field (line 280), before the `Runtime management` comment (line 282):

```go
	// Speculative decoding / draft model
	DraftModelID   string `json:"draftModelId"`   // Model ID of draft model (resolved to path by handler)
	DraftMaxTokens int    `json:"draftMaxTokens"`  // Max draft tokens per speculation round (default 16)
	DraftModelPath string `json:"-"`               // Resolved draft model file path (set by handler, not from JSON)
```

- [ ] **Step 2: Add draft fields to `backend.LoadRequest` in `internal/service/model/backend/backend.go`**

After `Devices` field (line 84), add:

```go
	DraftModelPath string // Resolved path to draft model GGUF
	DraftMaxTokens int    // Max draft tokens per speculation round
```

- [ ] **Step 3: Add draft fields to `LlamacppLoadParams` in `internal/service/model/backend/backend.go`**

After `RopeFreqScale` field (line 172), add:

```go
	DraftModelPath string
	DraftMaxTokens int
```

- [ ] **Step 4: Run `make build` to verify compilation**

Run: `make build`
Expected: compiles successfully (unused fields are not errors in Go)

- [ ] **Step 5: Commit**

```bash
git add internal/service/model/types.go internal/service/model/backend/backend.go
git commit -m "feat: add draft model fields to LoadRequest and backend types"
```

---

### Task 2: Backend — Build draft model CLI flags

**Files:**
- Modify: `internal/service/model/backend/llamacpp.go:293-301`
- Modify: `internal/service/model/load.go:370-428`

- [ ] **Step 1: Add `--draft` and `--draft-max-n` flags in `BuildStartConfig()`**

In `internal/service/model/backend/llamacpp.go`, insert after the RoPE scaling block (after line 298, before the "Build command string" comment at line 300):

```go
	// Speculative decoding / draft model
	if p.DraftModelPath != "" {
		args = append(args, "--draft", p.DraftModelPath)
	}
	if p.DraftMaxTokens > 0 {
		args = append(args, "--draft-max-n", strconv.Itoa(p.DraftMaxTokens))
	}
```

- [ ] **Step 2: Map draft fields in `toBackendLoadRequest()`**

In `internal/service/model/load.go`, inside the `default` case (line 371-427), add these two fields to the `LlamacppLoadParams` struct literal after `RopeFreqScale`:

```go
			DraftModelPath:  req.DraftModelPath,
			DraftMaxTokens:  req.DraftMaxTokens,
```

Also, at the top of `toBackendLoadRequest()` (around line 354-361), add `DraftModelPath` and `DraftMaxTokens` to the `backend.LoadRequest` struct:

```go
		DraftModelPath: req.DraftModelPath,
		DraftMaxTokens: req.DraftMaxTokens,
```

- [ ] **Step 3: Run `make build` to verify compilation**

Run: `make build`
Expected: compiles successfully

- [ ] **Step 4: Commit**

```bash
git add internal/service/model/backend/llamacpp.go internal/service/model/load.go
git commit -m "feat: build --draft and --draft-max-n CLI flags for speculative decoding"
```

---

### Task 3: Backend — Handler: resolve draft model ID and validate architecture

**Files:**
- Modify: `internal/server/handlers_model.go:92-154`

- [ ] **Step 1: Add imports to `handlers_model.go`**

Add to the import block:

```go
	"os"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/gguf"
```

(`"os"` is needed for file existence check; `gguf` for architecture parsing.)

- [ ] **Step 2: Add draft model resolution logic in `HandleLoadModel()`**

In `handlers_model.go`, insert the following block after `req.ModelID = id` (line 105) and before the default-value assignments (line 107):

```go
	// Resolve draft model ID to path and validate architecture
	if req.DraftModelID != "" {
		if req.DraftModelID == id {
			api.BadRequest(c, "Draft模型不能与主模型相同")
			return
		}
		draftModel, exists := s.modelMgr.GetModel(req.DraftModelID)
		if !exists {
			api.BadRequest(c, fmt.Sprintf("Draft模型未找到: %s", req.DraftModelID))
			return
		}
		draftPath := draftModel.Path
		if len(draftModel.ShardFiles) > 0 {
			draftPath = draftModel.ShardFiles[0]
		}
		if _, err := os.Stat(draftPath); err != nil {
			api.BadRequest(c, fmt.Sprintf("Draft模型文件不可访问: %s", draftPath))
			return
		}

		// Validate architecture compatibility
		mainModel, mainExists := s.modelMgr.GetModel(id)
		if !mainExists {
			api.BadRequest(c, "主模型未找到")
			return
		}
		mainPath := mainModel.Path
		if len(mainModel.ShardFiles) > 0 {
			mainPath = mainModel.ShardFiles[0]
		}

		mainArch := getArchitecture(mainPath)
		draftArch := getArchitecture(draftPath)
		if mainArch == "" || draftArch == "" {
			api.BadRequest(c, "无法读取模型架构信息，请确保模型文件有效")
			return
		}
		if !strings.EqualFold(mainArch, draftArch) {
			api.BadRequest(c, fmt.Sprintf("Draft模型架构(%s)与主模型架构(%s)不匹配", draftArch, mainArch))
			return
		}

		req.DraftModelPath = draftPath
		logger.Infof("draft model resolved: modelId=%s, draftModelId=%s, draftPath=%s, arch=%s", id, req.DraftModelID, draftPath, draftArch)
	}
```

- [ ] **Step 3: Add `getArchitecture` helper function**

Add a helper function at the end of `handlers_model.go` (or near the bottom, after the handler methods):

```go
func getArchitecture(modelPath string) string {
	parser, err := gguf.NewParser(modelPath)
	if err != nil {
		return ""
	}
	defer parser.Close()
	return parser.GetArchitecture()
}
```

- [ ] **Step 4: Add `"strings"` to imports if not already present**

Check if `"strings"` is in the import block. If not, add it. Also ensure `"fmt"` is imported.

- [ ] **Step 5: Run `make build` to verify compilation**

Run: `make build`
Expected: compiles successfully

- [ ] **Step 6: Commit**

```bash
git add internal/server/handlers_model.go
git commit -m "feat: resolve draft model ID to path and validate architecture in handler"
```

---

### Task 4: Frontend — Add draft model types

**Files:**
- Modify: `web/src/types/model.ts`

- [ ] **Step 1: Add draft fields to `LoadModelParams` interface**

In `web/src/types/model.ts`, after the `ropeFreqScale` field (line 184), add:

```typescript
  // Speculative decoding
  draftModelId?: string;       // Model ID of draft model
  draftMaxTokens?: number;     // Max draft tokens per speculation round
```

- [ ] **Step 2: Add enabled flags for draft fields**

In the `enabled` interface (inside `LoadModelParams`), after the `mmprojOffload` flag (line 286), add:

```typescript
    // Speculative decoding
    draftModel?: boolean;       // Enable/disable draft model selection
    draftMaxTokens?: boolean;   // Enable/disable draft max tokens
```

- [ ] **Step 3: Run `npm run type-check` to verify**

Run: `cd web && npm run type-check`
Expected: passes (new fields are optional)

- [ ] **Step 4: Commit**

```bash
git add web/src/types/model.ts
git commit -m "feat: add draft model fields to LoadModelParams TypeScript type"
```

---

### Task 5: Frontend — Add draft model UI to LoadModelDialog

**Files:**
- Modify: `web/src/features/models/components/LoadModelDialog.tsx`

This is the largest change. The dialog needs:
1. Import `useModels` hook
2. Add default values for draft fields in `useState` init (line 217) and `handleResetConfig` (line 609)
3. Add draft fields to `filterEnabledParams` paramKeys array (line 545)
4. Add draft model selector in left column (after llama.cpp version, around line 937)
5. Add draft-max-n in right column advanced params (new section)

- [ ] **Step 1: Add `useModels` to imports**

In line 11, add `useModels` to the import from `@/features/models`:

```typescript
import { useGPUs, useModelCapabilities, useSetModelCapabilities, useLlamacppBackends, useEstimateVRAM, useModelLoadConfig, useSaveModelLoadConfig, useDeleteModelLoadConfig, useAutoDetectCapabilities, useModels, type SystemGPUInfo, type LlamacppBackend } from '@/features/models';
```

- [ ] **Step 2: Add `useModels` hook call inside the component**

After the existing hook calls (look for `const { data: savedConfig... }` or similar hooks near the top of the component function), add:

```typescript
  const { data: modelsData } = useModels();
  const allModels = modelsData?.models ?? [];
```

- [ ] **Step 3: Compute draft candidate models**

After the `allModels` line, compute compatible draft models:

```typescript
  const mainArchitecture = modelPath ? undefined : undefined; // will be computed from model metadata
  const draftCandidates = allModels.filter(
    (m) => m.id !== modelId && m.metadata?.architecture && m.status === 'stopped'
  );
```

Note: We'll refine this to filter by architecture once we have the main model's metadata. The main model's architecture is available from `allModels` since we fetch all models.

- [ ] **Step 4: Add default values for draft fields in `useState`**

In the `useState<LoadModelParams>` initialization (around line 283, after `concurrencyLimit: 0`), add:

```typescript
    draftModelId: '',
    draftMaxTokens: 16,
```

In the `enabled` block (around line 340), add before the closing `}`:

```typescript
      draftModel: false,
      draftMaxTokens: false,
```

- [ ] **Step 5: Add same defaults in `handleResetConfig`**

Find the `handleResetConfig` function (around line 609) which has an identical default params object. Add the same `draftModelId: ''`, `draftMaxTokens: 16`, `draftModel: false`, `draftMaxTokens: false` fields there too.

- [ ] **Step 6: Add draft fields to `filterEnabledParams`**

In the `paramKeys` array (line 545-565), add at the end (before the closing `]`):

```typescript
      'draftModelId', 'draftMaxTokens',
```

- [ ] **Step 7: Add draft model selector in left column**

After the Llama.cpp version selector section (after line 937, before the capabilities section starting around line 940), insert:

```tsx
                {/* Draft Model / Speculative Decoding */}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <label className="text-xs font-medium text-muted-foreground">推测解码 (Draft 模型)</label>
                    <Switch
                      checked={params.enabled?.draftModel ?? false}
                      onCheckedChange={(checked) =>
                        setParams(prev => ({
                          ...prev,
                          enabled: { ...prev.enabled, draftModel: checked },
                          draftModelId: checked ? prev.draftModelId : '',
                        }))
                      }
                    />
                  </div>
                  {params.enabled?.draftModel && (
                    <Select
                      value={params.draftModelId || '_none'}
                      onValueChange={(val) =>
                        setParams(prev => ({ ...prev, draftModelId: val === '_none' ? '' : val }))
                      }
                    >
                      <SelectTrigger className="h-8 text-xs">
                        <SelectValue placeholder="选择 Draft 模型" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="_none">不使用</SelectItem>
                        {(() => {
                          const mainModel = allModels.find(m => m.id === modelId);
                          const mainArch = mainModel?.metadata?.architecture;
                          if (!mainArch) return null;
                          const compatible = allModels.filter(
                            m => m.id !== modelId
                              && m.metadata?.architecture === mainArch
                              && m.status === 'stopped'
                          );
                          if (compatible.length === 0) {
                            return <SelectItem value="_empty" disabled>没有找到同架构的候选模型</SelectItem>;
                          }
                          return compatible.map(m => (
                            <SelectItem key={m.id} value={m.id}>
                              {m.displayName} ({(m.totalSize ?? m.size / 1024 / 1024 / 1024).toFixed(1)}GB)
                            </SelectItem>
                          ));
                        })()}
                      </SelectContent>
                    </Select>
                  )}
                </div>
```

- [ ] **Step 8: Add draft-max-n in right column advanced params**

In the right column, before the last section "其他参数" (around line 1610), insert a new section:

```tsx
                {/* Speculative decoding */}
                <div className="space-y-3">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                    推测解码
                  </h4>
                  <div className="flex items-center gap-2">
                    <Switch
                      checked={params.enabled?.draftMaxTokens ?? false}
                      onCheckedChange={(checked) =>
                        setParams(prev => ({
                          ...prev,
                          enabled: { ...prev.enabled, draftMaxTokens: checked },
                        }))
                      }
                    />
                    <div className="flex-1">
                      <div className="flex items-center gap-1.5">
                        <label className="text-xs font-medium">Draft 最大 token 数</label>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Info className="h-3 w-3 text-muted-foreground cursor-help" />
                          </TooltipTrigger>
                          <TooltipContent>每轮推测解码中 draft 模型生成的最大 token 数（默认 16）</TooltipContent>
                        </Tooltip>
                      </div>
                    </div>
                    <NumberInput
                      value={params.draftMaxTokens ?? 16}
                      onChange={(val) => setParams(prev => ({ ...prev, draftMaxTokens: val }))}
                      disabled={!params.enabled?.draftMaxTokens}
                      min={1}
                      max={256}
                      step={1}
                      placeholder="16"
                      className="w-20 h-7 text-xs"
                    />
                  </div>
                </div>
```

- [ ] **Step 9: Run lint and type-check**

Run: `cd web && npm run lint:fix && npm run type-check`
Expected: passes

- [ ] **Step 10: Commit**

```bash
git add web/src/features/models/components/LoadModelDialog.tsx
git commit -m "feat: add draft model selector and max tokens UI to LoadModelDialog"
```

---

### Task 6: Verification — Build and lint everything

**Files:** None (verification only)

- [ ] **Step 1: Run backend build**

Run: `make build`
Expected: compiles successfully

- [ ] **Step 2: Run backend lint**

Run: `make fmt && make lint`
Expected: passes

- [ ] **Step 3: Run frontend type-check and lint**

Run: `cd web && npm run type-check && npm run lint:fix`
Expected: passes

- [ ] **Step 4: Run frontend tests**

Run: `cd web && npm run test`
Expected: passes (existing tests should not be affected)

- [ ] **Step 5: Final commit if any formatting fixes were applied**

```bash
git add -A
git commit -m "chore: lint and formatting fixes" || true
```
