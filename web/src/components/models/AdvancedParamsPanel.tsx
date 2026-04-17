import type { LoadModelParams } from '@/types';
import { NumberInput } from './NumberInput';
import { SelectInput } from './SelectInput';
import { ParamControl } from './ParamControl';

interface AdvancedParamsPanelProps {
  params: LoadModelParams;
  onParamsChange: (params: LoadModelParams) => void;
  isLoading: boolean;
  isParamEnabled: (paramKey: string) => boolean;
  getInputDisabled: (paramKey: string) => boolean;
  activeTooltip: string | null;
  onSetActiveTooltip: React.Dispatch<React.SetStateAction<string | null>>;
  onToggleEnabled: (paramKey: string) => void;
}

export function AdvancedParamsPanel({
  params,
  onParamsChange,
  isLoading,
  isParamEnabled,
  getInputDisabled,
  activeTooltip,
  onSetActiveTooltip,
  onToggleEnabled,
}: AdvancedParamsPanelProps) {
  const renderHelpButton = (paramKey: string, showToggle = true) => (
    <ParamControl
      paramKey={paramKey}
      showToggle={showToggle}
      isLoading={isLoading}
      isEnabled={isParamEnabled(paramKey)}
      onToggleEnabled={onToggleEnabled}
      activeTooltip={activeTooltip}
      onSetActiveTooltip={onSetActiveTooltip}
    />
  );

  return (
    <div className="flex-1 space-y-4 overflow-y-auto pr-2 min-h-0" aria-label="高级参数区域">
      <h3 className="text-sm font-semibold text-foreground pb-2 border-b border-border">
        高级参数
      </h3>

      {/* 上下文与加速 */}
      <div className="space-y-3">
        <h4 className="text-xs font-semibold text-muted-foreground uppercase">
          上下文与加速
        </h4>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --ctx-size
              {renderHelpButton('ctxSize')}
            </label>
            <NumberInput
              value={params.ctxSize}
              onChange={(v) => onParamsChange({ ...params, ctxSize: v })}
              disabled={getInputDisabled('ctxSize')}
              min={0}
              max={131072}
              step={1}
              placeholder="8192"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              Flash Attention
              {renderHelpButton('flashAttention')}
            </label>
            <SelectInput
              value={params.flashAttention ? 'on' : 'off'}
              onChange={(e) => onParamsChange({ ...params, flashAttention: e.target.value === 'on' })}
              disabled={getInputDisabled('flashAttention')}
            >
              <option value="on">on</option>
              <option value="off">off</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --no-mmap
              {renderHelpButton('noMmap')}
            </label>
            <SelectInput
              value={params.noMmap ? 'true' : 'false'}
              onChange={(e) => onParamsChange({ ...params, noMmap: e.target.value === 'true' })}
              disabled={getInputDisabled('noMmap')}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              锁定物理内存
              {renderHelpButton('lockMemory')}
            </label>
            <SelectInput
              value={params.lockMemory ? 'true' : 'false'}
              onChange={(e) => onParamsChange({ ...params, lockMemory: e.target.value === 'true' })}
              disabled={getInputDisabled('lockMemory')}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --embedding
              {renderHelpButton('embedding')}
            </label>
            <SelectInput
              value={params.embedding ? 'true' : 'false'}
              onChange={(e) => onParamsChange({ ...params, embedding: e.target.value === 'true' })}
              disabled={getInputDisabled('embedding')}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --reranking
              {renderHelpButton('reranking')}
            </label>
            <SelectInput
              value={params.reranking ? 'true' : 'false'}
              onChange={(e) => onParamsChange({ ...params, reranking: e.target.value === 'true' })}
              disabled={getInputDisabled('reranking')}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --gpu-layers
              {renderHelpButton('gpuLayers')}
            </label>
            <NumberInput
              value={params.gpuLayers}
              onChange={(v) => onParamsChange({ ...params, gpuLayers: v })}
              disabled={getInputDisabled('gpuLayers')}
              min={-1}
              max={999}
              step={1}
              placeholder="-1 表示全部"
              allowMinusOne={true}
            />
          </div>
        </div>
      </div>

      {/* 采样参数 */}
      <div className="space-y-3">
        <h4 className="text-xs font-semibold text-muted-foreground uppercase">
          采样参数
        </h4>
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --temp
              {renderHelpButton('temperature')}
            </label>
            <NumberInput
              value={params.temperature}
              onChange={(v) => onParamsChange({ ...params, temperature: v })}
              disabled={getInputDisabled('temperature')}
              min={0}
              max={2}
              step={0.1}
              placeholder="0.7"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              Top-P
              {renderHelpButton('topP')}
            </label>
            <NumberInput
              value={params.topP}
              onChange={(v) => onParamsChange({ ...params, topP: v })}
              disabled={getInputDisabled('topP')}
              min={0}
              max={1}
              step={0.05}
              placeholder="0.95"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              Top-K
              {renderHelpButton('topK')}
            </label>
            <NumberInput
              value={params.topK}
              onChange={(v) => onParamsChange({ ...params, topK: v })}
              disabled={getInputDisabled('topK')}
              min={1}
              max={1000}
              step={1}
              placeholder="40"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              Min-P
              {renderHelpButton('minP')}
            </label>
            <NumberInput
              value={params.minP}
              onChange={(v) => onParamsChange({ ...params, minP: v })}
              disabled={getInputDisabled('minP')}
              min={0}
              max={1}
              step={0.01}
              placeholder="0.05"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --repeat-penalty
              {renderHelpButton('repeatPenalty')}
            </label>
            <NumberInput
              value={params.repeatPenalty}
              onChange={(v) => onParamsChange({ ...params, repeatPenalty: v })}
              disabled={getInputDisabled('repeatPenalty')}
              min={0}
              max={2}
              step={0.05}
              placeholder="1.1"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --presence-penalty
              {renderHelpButton('presencePenalty')}
            </label>
            <NumberInput
              value={params.presencePenalty}
              onChange={(v) => onParamsChange({ ...params, presencePenalty: v })}
              disabled={getInputDisabled('presencePenalty')}
              min={0}
              max={2}
              step={0.1}
              placeholder="0.0"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --frequency-penalty
              {renderHelpButton('frequencyPenalty')}
            </label>
            <NumberInput
              value={params.frequencyPenalty}
              onChange={(v) => onParamsChange({ ...params, frequencyPenalty: v })}
              disabled={getInputDisabled('frequencyPenalty')}
              min={0}
              max={2}
              step={0.1}
              placeholder="0.0"
            />
          </div>
        </div>
      </div>

      {/* 批处理与并发 */}
      <div className="space-y-3">
        <h4 className="text-xs font-semibold text-muted-foreground uppercase">
          批处理与并发
        </h4>
        <div className="grid grid-cols-4 gap-3">
          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --batch-size
              {renderHelpButton('batchSize')}
            </label>
            <NumberInput
              value={params.batchSize}
              onChange={(v) => onParamsChange({ ...params, batchSize: v })}
              disabled={getInputDisabled('batchSize')}
              min={64}
              max={16384}
              step={64}
              placeholder="4096"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --ubatch-size
              {renderHelpButton('uBatchSize')}
            </label>
            <NumberInput
              value={params.uBatchSize}
              onChange={(v) => onParamsChange({ ...params, uBatchSize: v })}
              disabled={getInputDisabled('uBatchSize')}
              min={64}
              max={8192}
              step={64}
              placeholder="512"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --parallel(并发槽数)
              {renderHelpButton('parallelSlots')}
            </label>
            <NumberInput
              value={params.parallelSlots}
              onChange={(v) => onParamsChange({ ...params, parallelSlots: v })}
              disabled={getInputDisabled('parallelSlots')}
              min={1}
              max={128}
              step={1}
              placeholder="4"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              线程数
              {renderHelpButton('threads')}
            </label>
            <NumberInput
              value={params.threads}
              onChange={(v) => onParamsChange({ ...params, threads: v })}
              disabled={getInputDisabled('threads')}
              min={-1}
              max={256}
              step={1}
              placeholder="-1 表示自动"
              allowMinusOne={true}
            />
          </div>
        </div>
      </div>

      {/* KV缓存 */}
      <div className="space-y-3">
        <h4 className="text-xs font-semibold text-muted-foreground uppercase">
          KV缓存
        </h4>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --cache-ram
              {renderHelpButton('kvCacheSize')}
            </label>
            <NumberInput
              value={params.kvCacheSize}
              onChange={(v) => onParamsChange({ ...params, kvCacheSize: v })}
              disabled={getInputDisabled('kvCacheSize')}
              min={0}
              max={131072}
              step={1}
              placeholder="8192"
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --kv-unified
              {renderHelpButton('kvCacheUnified')}
            </label>
            <SelectInput
              value={params.kvCacheUnified ? 'true' : 'false'}
              onChange={(e) => onParamsChange({ ...params, kvCacheUnified: e.target.value === 'true' })}
              disabled={getInputDisabled('kvCacheUnified')}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </SelectInput>
          </div>

          <div>
            <label className="text-xs font-medium text-foreground mb-1">
              -ctk
            </label>
            <SelectInput
              value={params.kvCacheTypeK || 'f16'}
              onChange={(e) => onParamsChange({ ...params, kvCacheTypeK: e.target.value })}
              disabled={getInputDisabled('kvCacheTypeK')}
            >
              <option value="f32">f32</option>
              <option value="f16">f16 (默认)</option>
              <option value="bf16">bf16</option>
              <option value="q8_0">q8_0</option>
              <option value="q5_0">q5_0</option>
              <option value="q5_1">q5_1</option>
              <option value="q4_0">q4_0</option>
              <option value="q4_1">q4_1</option>
              <option value="iq4_nl">iq4_nl</option>
            </SelectInput>
          </div>

          <div>
            <label className="text-xs font-medium text-foreground mb-1">
              -ctv
            </label>
            <SelectInput
              value={params.kvCacheTypeV || 'f16'}
              onChange={(e) => onParamsChange({ ...params, kvCacheTypeV: e.target.value })}
              disabled={getInputDisabled('kvCacheTypeV')}
            >
              <option value="f32">f32</option>
              <option value="f16">f16 (默认)</option>
              <option value="bf16">bf16</option>
              <option value="q8_0">q8_0</option>
              <option value="q5_0">q5_0</option>
              <option value="q5_1">q5_1</option>
              <option value="q4_0">q4_0</option>
              <option value="q4_1">q4_1</option>
              <option value="iq4_nl">iq4_nl</option>
            </SelectInput>
          </div>
        </div>
      </div>

      {/* 其他参数 */}
      <div className="space-y-3">
        <h4 className="text-xs font-semibold text-muted-foreground uppercase">
          其他参数
        </h4>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --seed
              {renderHelpButton('seed')}
            </label>
            <NumberInput
              value={params.seed}
              onChange={(v) => onParamsChange({ ...params, seed: v })}
              disabled={getInputDisabled('seed')}
              min={-1}
              max={4294967295}
              step={1}
              placeholder="-1 表示随机"
              allowMinusOne={true}
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --n-predict
              {renderHelpButton('nPredict')}
            </label>
            <NumberInput
              value={params.nPredict}
              onChange={(v) => onParamsChange({ ...params, nPredict: v })}
              disabled={getInputDisabled('nPredict')}
              min={-1}
              max={65536}
              step={64}
              placeholder="-1 表示无限"
              allowMinusOne={true}
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --direct-io
              {renderHelpButton('directIo')}
            </label>
            <SelectInput
              value={params.directIo || 'default'}
              onChange={(e) => onParamsChange({ ...params, directIo: e.target.value })}
              disabled={getInputDisabled('directIo')}
            >
              <option value="default">default</option>
              <option value="true">true</option>
              <option value="false">false</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --no-webui
              {renderHelpButton('noWebUI')}
            </label>
            <SelectInput
              value={params.noWebUI ? 'true' : 'false'}
              onChange={(e) => onParamsChange({ ...params, noWebUI: e.target.value === 'true' })}
              disabled={getInputDisabled('noWebUI')}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --no-jinja
              {renderHelpButton('disableJinja')}
            </label>
            <SelectInput
              value={params.disableJinja ? 'true' : 'false'}
              onChange={(e) => onParamsChange({ ...params, disableJinja: e.target.value === 'true' })}
              disabled={getInputDisabled('disableJinja')}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --chat-template
              {renderHelpButton('chatTemplate')}
            </label>
            <SelectInput
              value={params.chatTemplate || ''}
              onChange={(e) => onParamsChange({ ...params, chatTemplate: e.target.value })}
              disabled={getInputDisabled('chatTemplate')}
            >
              <option value="">(自动 - 使用模型元数据)</option>
              {/* Bailing 系列 */}
              <option value="bailing">bailing</option>
              <option value="bailing-think">bailing-think</option>
              <option value="bailing2">bailing2</option>
              {/* ChatGLM 系列 */}
              <option value="chatglm3">chatglm3</option>
              <option value="chatglm4">chatglm4</option>
              {/* DeepSeek 系列 */}
              <option value="deepseek">deepseek</option>
              <option value="deepseek2">deepseek2</option>
              <option value="deepseek3">deepseek3</option>
              {/* Exaone 系列 */}
              <option value="exaone-moe">exaone-moe</option>
              <option value="exaone3">exaone3</option>
              <option value="exaone4">exaone4</option>
              {/* Hunyuan 系列 */}
              <option value="hunyuan-dense">hunyuan-dense</option>
              <option value="hunyuan-moe">hunyuan-moe</option>
              {/* Llama 系列 */}
              <option value="llama2">llama2</option>
              <option value="llama2-sys">llama2-sys</option>
              <option value="llama2-sys-bos">llama2-sys-bos</option>
              <option value="llama2-sys-strip">llama2-sys-strip</option>
              <option value="llama3">llama3</option>
              <option value="llama4">llama4</option>
              {/* Mistral 系列 */}
              <option value="mistral-v1">mistral-v1</option>
              <option value="mistral-v3">mistral-v3</option>
              <option value="mistral-v3-tekken">mistral-v3-tekken</option>
              <option value="mistral-v7">mistral-v7</option>
              <option value="mistral-v7-tekken">mistral-v7-tekken</option>
              {/* Phi 系列 */}
              <option value="phi3">phi3</option>
              <option value="phi4">phi4</option>
              {/* Vicuna 系列 */}
              <option value="vicuna">vicuna</option>
              <option value="vicuna-orca">vicuna-orca</option>
              {/* 其他模板 */}
              <option value="chatml">chatml</option>
              <option value="command-r">command-r</option>
              <option value="falcon3">falcon3</option>
              <option value="gemma">gemma</option>
              <option value="gigachat">gigachat</option>
              <option value="glmedge">glmedge</option>
              <option value="gpt-oss">gpt-oss</option>
              <option value="granite">granite</option>
              <option value="grok-2">grok-2</option>
              <option value="kimi-k2">kimi-k2</option>
              <option value="megrez">megrez</option>
              <option value="minicpm">minicpm</option>
              <option value="monarch">monarch</option>
              <option value="openchat">openchat</option>
              <option value="orion">orion</option>
              <option value="pangu-embedded">pangu-embedded</option>
              <option value="rwkv-world">rwkv-world</option>
              <option value="seed_oss">seed_oss</option>
              <option value="smolvlm">smolvlm</option>
              <option value="solar-open">solar-open</option>
              <option value="yandex">yandex</option>
              <option value="zephyr">zephyr</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --context-shift
              {renderHelpButton('contextShift')}
            </label>
            <SelectInput
              value={params.contextShift ? 'true' : 'false'}
              onChange={(e) => onParamsChange({ ...params, contextShift: e.target.value === 'true' })}
              disabled={getInputDisabled('contextShift')}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --reasoning
              {renderHelpButton('reasoning')}
            </label>
            <SelectInput
              value={params.reasoning || 'auto'}
              onChange={(e) => onParamsChange({ ...params, reasoning: e.target.value })}
              disabled={getInputDisabled('reasoning')}
            >
              <option value="auto">auto</option>
              <option value="on">on</option>
              <option value="off">off</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --reasoning-format
              {renderHelpButton('reasoningFormat')}
            </label>
            <SelectInput
              value={params.reasoningFormat || 'auto'}
              onChange={(e) => onParamsChange({ ...params, reasoningFormat: e.target.value })}
              disabled={getInputDisabled('reasoningFormat')}
            >
              <option value="auto">auto</option>
              <option value="deepseek">deepseek</option>
            </SelectInput>
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --reasoning-budget
              {renderHelpButton('reasoningBudget')}
            </label>
            <NumberInput
              value={params.reasoningBudget}
              onChange={(v) => onParamsChange({ ...params, reasoningBudget: v })}
              disabled={getInputDisabled('reasoningBudget')}
              min={-1}
              max={100000}
              step={256}
              placeholder="-1 无限制"
              allowMinusOne={true}
            />
          </div>

          <div>
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              --no-mmproj-offload
              {renderHelpButton('mmprojOffload')}
            </label>
            <SelectInput
              value={params.mmprojOffload ? 'true' : 'false'}
              onChange={(e) => onParamsChange({ ...params, mmprojOffload: e.target.value === 'true' })}
              disabled={getInputDisabled('mmprojOffload')}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </SelectInput>
          </div>

          <div className="col-span-2">
            <label className="flex items-center text-xs font-medium text-foreground mb-1">
              其他参数
              {renderHelpButton('extraArgs')}
            </label>
            <textarea
              value={params.extraArgs || ''}
              onChange={(e) => onParamsChange({ ...params, extraArgs: e.target.value })}
              disabled={getInputDisabled('extraArgs')}
              rows={2}
              placeholder="例如: --timeout 30 --grp-attn-n 8"
              className="w-full px-2 py-1.5 text-sm border-2 border-border rounded-md bg-input text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 resize-y"
            />
            <p className="text-xs text-muted-foreground mt-1">
              输入额外的命令行参数，用空格分隔
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
