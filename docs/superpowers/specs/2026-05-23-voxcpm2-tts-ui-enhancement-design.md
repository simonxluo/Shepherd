# VoxCPM2 TTS UI Enhancement Design

## Overview

完善 VoxCPM2 TTS 面板：将语言选择从自由文本改为下拉框（30+ 语言），补全高级配置参数（cfg_cutoff_ratio, sway_sampling_coef），通过 extra_params 透传到后端。

## Background

- VoxCPM2 官方支持 30 种语言 + 9 种中文方言
- 当前 UI 的语言字段是自由文本 Input，用户需要知道语言代码
- 高级设置缺少 `cfg_cutoff_ratio` 和 `sway_sampling_coef` 两个模型参数
- 后端 VoxCPM2 路径当前自动检测语言，不使用 language 参数，但 UI 下拉有文档/提示价值
- 新增的 cfg_cutoff_ratio/sway_sampling_coef 通过 `extra_params` JSON 字段透传给 vllm-omni 后端

## Changes

### 1. Frontend: Language Dropdown

**File**: `web/src/features/tts/plugins/voxcpm2/VoxCPM2Panel.tsx`

- 将语言 `<Input>` 替换为 `<Select>` 下拉框
- 定义语言常量数组，包含值和显示标签
- 默认值改为空字符串（表示 Auto/自动检测），用户可手动选择语言
- 保留 i18n key 兼容

**语言列表**（按分组）：
- Auto（自动检测）— 默认选项
- 30 种官方语言（按英文字母排序）
- 9 种中文方言

值格式使用全名（如 "Chinese", "English"），与 vllm-omni `_TTS_LANGUAGES` 保持一致。中文方言作为独立选项。

### 2. Frontend: Advanced Configuration

**File**: `web/src/features/tts/plugins/voxcpm2/VoxCPM2Panel.tsx`

新增高级参数：

| 参数 | 类型 | 范围 | 默认值 | 说明 |
|------|------|------|--------|------|
| cfg_cutoff_ratio | Slider | 0-1 | 1.0 | CFG 截止步数比例，控制 CFG 在哪一步停止 |
| sway_sampling_coef | Input(number) | >= 0 | 1.0 | Sway 采样系数，影响采样分布 |

这些参数仅在 features flag 启用时显示（复用现有 feature gate 模式）。

### 3. Frontend: Plugin Feature Flags

**File**: `web/src/features/tts/plugins/voxcpm2/index.ts`, `web/src/features/tts/hooks.ts`

在 `TTSModelFeatures` 中新增：
- `supportsCfgCutoffRatio: boolean` — 默认 true for VoxCPM
- `supportsSwaySampling: boolean` — 默认 true for VoxCPM

### 4. Frontend: Data Flow

**File**: `web/src/features/tts/hooks.ts`

- `TTSConfig` 新增 `cfgCutoffRatio?: string` 和 `swaySamplingCoef?: string`
- `extractTTSConfig` 提取新字段
- `handleGenerate` 中构建 `extra_params` 对象，将新参数透传

**File**: `web/src/features/tts/plugins/voxcpm2/VoxCPM2Panel.tsx`

- `getCurrentConfig` / `handleLoadConfig` / `useEffect` (config restore) 中加入新字段

### 5. Backend: extra_params Proxy

**File**: `internal/handler/openai/audio_handler.go`

- 确保 `extra_params` 字段从请求体透传到 vllm-omni 后端
- 检查当前是否已有 extra_params 的代理逻辑

### 6. i18n

**Files**: `web/src/locales/zh-CN/*.json`, `web/src/locales/en-US/*.json`

新增翻译 key：
- `tts.languageAuto`: "自动检测" / "Auto Detect"
- `tts.cfgCutoffRatio`: "CFG 截止比例" / "CFG Cutoff Ratio"
- `tts.swaySamplingCoef`: "Sway 采样系数" / "Sway Sampling Coefficient"

## Language Constant

```typescript
const VOXCPM2_LANGUAGES = [
  // Auto detect
  { value: '', label: 'tts.languageAuto', fallback: 'Auto Detect' },
  // 30 official languages
  { value: 'Arabic', label: 'Arabic' },
  { value: 'Burmese', label: 'Burmese' },
  { value: 'Chinese', label: 'Chinese' },
  { value: 'Danish', label: 'Danish' },
  { value: 'Dutch', label: 'Dutch' },
  { value: 'English', label: 'English' },
  { value: 'Finnish', label: 'Finnish' },
  { value: 'French', label: 'French' },
  { value: 'German', label: 'German' },
  { value: 'Greek', label: 'Greek' },
  { value: 'Hebrew', label: 'Hebrew' },
  { value: 'Hindi', label: 'Hindi' },
  { value: 'Indonesian', label: 'Indonesian' },
  { value: 'Italian', label: 'Italian' },
  { value: 'Japanese', label: 'Japanese' },
  { value: 'Khmer', label: 'Khmer' },
  { value: 'Korean', label: 'Korean' },
  { value: 'Lao', label: 'Lao' },
  { value: 'Malay', label: 'Malay' },
  { value: 'Norwegian', label: 'Norwegian' },
  { value: 'Polish', label: 'Polish' },
  { value: 'Portuguese', label: 'Portuguese' },
  { value: 'Russian', label: 'Russian' },
  { value: 'Spanish', label: 'Spanish' },
  { value: 'Swahili', label: 'Swahili' },
  { value: 'Swedish', label: 'Swedish' },
  { value: 'Tagalog', label: 'Tagalog' },
  { value: 'Thai', label: 'Thai' },
  { value: 'Turkish', label: 'Turkish' },
  { value: 'Vietnamese', label: 'Vietnamese' },
  // 9 Chinese dialects
  { value: '四川话', label: '四川话 (Sichuanese)' },
  { value: '粤语', label: '粤语 (Cantonese)' },
  { value: '吴语', label: '吴语 (Wu)' },
  { value: '东北话', label: '东北话 (Northeastern)' },
  { value: '河南话', label: '河南话 (Henan)' },
  { value: '陕西方言', label: '陕西方言 (Shaanxi)' },
  { value: '山东话', label: '山东话 (Shandong)' },
  { value: '天津话', label: '天津话 (Tianjin)' },
  { value: '闽南话', label: '闽南话 (Min Nan)' },
];
```

## Implementation Order

1. 定义语言常量数组（VoxCPM2Panel.tsx 或独立文件）
2. 更新 TTSModelFeatures 和 plugin features
3. 更新 TTSConfig 类型
4. 更新 VoxCPM2Panel UI（语言下拉 + 新高级参数）
5. 更新数据流（config save/load/restore/generate）
6. 后端 extra_params 透传检查
7. i18n 翻译
8. 测试

## Testing

- 手动测试：语言下拉选择、配置保存/恢复、生成请求
- 验证 extra_params 正确传递到后端请求体
- 验证现有功能（Voice Cloning、Ultimate Cloning、Streaming）无回归

## Scope

- 不改动通用 TTS 插件（generic plugin）
- 不改动后端模型推理逻辑
- 语言下拉仅影响 VoxCPM2 面板
