# Button Style Audit - LoadModelDialog

## Summary

This audit analyzes all button styles in `LoadModelDialog.tsx` to identify inconsistencies.

## Button Locations

| Button Type | Lines | Description |
|-------------|-------|-------------|
| Preset Buttons | 1020-1042 | 快速, 均衡, 性能, 极致 |
| Reset Button | 1046-1062 | 重置按钮 |
| Auto-detect Button | 1134-1177 | 自动检测能力按钮 |
| Footer Buttons | 2014-2102 | 取消, 估算显存, 保存配置, 开始加载 |

## Style Comparison

### Inline Buttons (preset, reset, auto-detect)

| Property | Preset Buttons | Reset Button | Auto-detect Button |
|----------|---------------|--------------|-------------------|
| padding-x | `px-3` | `px-3` | `px-3` |
| padding-y | `py-1.5` | `py-1.5` | `py-1.5` |
| font-size | `text-sm` | `text-sm` | `text-sm` |
| font-weight | `font-medium` | `font-medium` | `font-medium` |
| border-radius | `rounded-md` | `rounded-md` | `rounded-md` |
| **height** | ❌ No fixed height | ❌ No fixed height | ❌ No fixed height |
| display | block | `flex items-center gap-1.5` | `flex items-center gap-1.5` |

### Footer Buttons (Button component)

| Button | Variant | Has Icon | Notes |
|--------|---------|----------|-------|
| 取消 | `outline` | No | Standard Button |
| 估算显存 | `outline` | No | Standard Button |
| 保存配置 | `secondary` | Yes (Save icon) | Dynamic color based on status |
| 开始加载 | default | Yes (Loader2 when loading) | Submit button |

## Identified Issues

### Issue 1: No Fixed Height for Inline Buttons
**Severity**: Medium
**Impact**: Buttons may have slightly different heights due to content

**Current**:
```tsx
"px-3 py-1.5 text-sm font-medium rounded-md"
```

**Recommended Fix**:
```tsx
"px-3 py-1.5 h-[34px] text-sm font-medium rounded-md flex items-center"
```

### Issue 2: Inconsistent Display Property
**Severity**: Low
**Impact**: Preset buttons don't use flex, which may cause icon alignment issues

**Current**:
- Preset buttons: No `flex items-center`
- Reset/Auto-detect: `flex items-center gap-1.5`

**Recommended Fix**: Add `flex items-center` to all inline buttons for consistency.

### Issue 3: Footer Button Component vs Inline Buttons
**Severity**: Low
**Impact**: Footer buttons use `<Button>` component which has its own sizing

**Note**: Footer buttons use the `Button` component from `@/components/ui/Button`. This is acceptable as they have different visual hierarchy (primary actions).

## Recommended Changes

1. **Add fixed height `h-[34px]`** to all inline buttons
2. **Add `flex items-center`** to preset buttons for icon alignment
3. **Keep footer buttons as-is** - they use the Button component intentionally

## Files to Modify

- `web/src/components/models/LoadModelDialog.tsx`
  - Lines 1026-1037: Add `h-[34px] flex items-center` to preset buttons
  - Lines 1050-1056: Add `h-[34px]` to reset button (already has flex)
  - Lines 1162-1168: Add `h-[34px]` to auto-detect button (already has flex)
