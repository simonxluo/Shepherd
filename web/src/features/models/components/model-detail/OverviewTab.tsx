import { useState, useMemo } from 'react';
import { Copy, Check } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { formatBytes } from '@/lib/utils';
import type { Model } from '@/types';

interface OverviewTabProps {
  model: Model;
}

function DetailRow({ label, value, mono, onCopy }: {
  label: string;
  value: string | number | undefined | null;
  mono?: boolean;
  onCopy?: boolean;
}) {
  const [copied, setCopied] = useState(false);
  const displayValue = value === undefined || value === null || value === '' ? '-' : String(value);

  const handleCopy = () => {
    if (displayValue !== '-') {
      navigator.clipboard.writeText(displayValue);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="flex items-center justify-between py-2 px-3 hover:bg-muted/50 rounded-md group min-h-[36px]">
      <span className="text-sm text-muted-foreground font-medium shrink-0 mr-4">{label}</span>
      <div className="flex items-center gap-2 min-w-0">
        <span className={`text-sm text-foreground truncate ${mono ? 'font-mono' : ''}`} title={displayValue}>
          {displayValue}
        </span>
        {onCopy && displayValue !== '-' && (
          <button onClick={handleCopy} className="opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
            {copied ? <Check className="w-3.5 h-3.5 text-green-500" /> : <Copy className="w-3.5 h-3.5 text-muted-foreground hover:text-foreground" />}
          </button>
        )}
      </div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mb-4">
      <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2 px-1">{title}</h4>
      <div className="bg-muted/30 rounded-lg overflow-hidden divide-y divide-border/50">
        {children}
      </div>
    </div>
  );
}

export function OverviewTab({ model }: OverviewTabProps) {
  const { t } = useTranslation();
  const meta = model.metadata;

  const hasStructure = useMemo(() => {
    return !!(meta.contextLength || meta.embeddingLength || meta.feedForwardLength ||
      meta.blockCount || meta.headCount || meta.headCountKV);
  }, [meta]);

  const hasTokenizer = useMemo(() => {
    return !!(meta.tokenizerModel || meta.tokenCount);
  }, [meta]);

  const hasRope = useMemo(() => {
    return !!(meta.ropeDim || meta.ropeFreqBase || meta.ropeFreqScale);
  }, [meta]);

  const statusText = model.status === 'running' ? t('modelDetail.values.running', '运行中') :
    model.status === 'loading' ? t('modelDetail.values.loading', '加载中') :
    model.status === 'stopped' ? t('modelDetail.values.stopped', '已停止') :
    model.status === 'unloading' ? t('modelDetail.values.unloading', '卸载中') :
    t('modelDetail.values.error', '错误');

  return (
    <div className="space-y-2 overflow-y-auto pr-1">
      {/* Basic Info */}
      <Section title={t('modelDetail.sections.basicInfo', '基本信息')}>
        <DetailRow label={t('modelDetail.fields.architecture', '架构')} value={meta.architecture} />
        <DetailRow label={t('modelDetail.fields.quantization', '量化')} value={meta.quantization || meta.fileTypeDescriptor} />
        <DetailRow label={t('modelDetail.fields.parameters', '参数量')} value={meta.parameters ? `${(meta.parameters / 1e9).toFixed(2)}B` : undefined} />
        <DetailRow label={t('modelDetail.fields.size', '大小')} value={formatBytes(model.totalSize ?? model.size)} />
        {model.shardCount && model.shardCount > 1 && (
          <DetailRow label={t('modelDetail.fields.shards', '分片')} value={`${model.shardCount} ${t('modelDetail.values.files', '个文件')}`} />
        )}
        <DetailRow label={t('modelDetail.fields.path', '路径')} value={model.path} mono onCopy />
        <DetailRow label={t('modelDetail.fields.id', 'ID')} value={model.id} mono onCopy />
        {model.mmprojPath && (
          <DetailRow label={t('modelDetail.fields.mmprojPath', '视觉模型')} value={model.mmprojPath} mono onCopy />
        )}
        {meta.author && <DetailRow label={t('modelDetail.fields.author', '作者')} value={meta.author} />}
        {meta.url && <DetailRow label={t('modelDetail.fields.url', '来源')} value={meta.url} onCopy />}
        {meta.license && <DetailRow label={t('modelDetail.fields.license', '许可证')} value={meta.license} />}
      </Section>

      {/* Model Structure */}
      {hasStructure && (
        <Section title={t('modelDetail.sections.architecture', '模型结构')}>
          {meta.contextLength ? <DetailRow label={t('modelDetail.fields.contextLength', '上下文长度')} value={meta.contextLength.toLocaleString()} /> : null}
          {meta.embeddingLength ? <DetailRow label={t('modelDetail.fields.embeddingLength', '嵌入维度')} value={meta.embeddingLength.toLocaleString()} /> : null}
          {meta.feedForwardLength ? <DetailRow label={t('modelDetail.fields.feedForwardLength', '前馈维度')} value={meta.feedForwardLength.toLocaleString()} /> : null}
          {meta.blockCount ? <DetailRow label={t('modelDetail.fields.blockCount', '层数')} value={meta.blockCount} /> : null}
          {meta.headCount ? <DetailRow label={t('modelDetail.fields.headCount', '注意力头数')} value={meta.headCount} /> : null}
          {meta.headCountKV ? <DetailRow label={t('modelDetail.fields.headCountKV', 'KV头数')} value={meta.headCountKV} /> : null}
          {meta.layerNormRmsEps ? <DetailRow label={t('modelDetail.fields.layerNormRmsEps', 'RMS Epsilon')} value={meta.layerNormRmsEps.toExponential(2)} /> : null}
        </Section>
      )}

      {/* Tokenizer */}
      {hasTokenizer && (
        <Section title={t('modelDetail.sections.tokenizer', 'Tokenizer')}>
          {meta.tokenizerModel && <DetailRow label={t('modelDetail.fields.tokenizerModel', '模型')} value={meta.tokenizerModel} />}
          {meta.tokenCount ? <DetailRow label={t('modelDetail.fields.tokenCount', '词表大小')} value={meta.tokenCount.toLocaleString()} /> : null}
          {meta.bosTokenId !== undefined && meta.bosTokenId !== 0 && <DetailRow label={t('modelDetail.fields.bosTokenId', 'BOS Token')} value={meta.bosTokenId} />}
          {meta.eosTokenId !== undefined && meta.eosTokenId !== 0 && <DetailRow label={t('modelDetail.fields.eosTokenId', 'EOS Token')} value={meta.eosTokenId} />}
          {meta.padTokenId !== undefined && meta.padTokenId !== 0 && <DetailRow label={t('modelDetail.fields.padTokenId', 'PAD Token')} value={meta.padTokenId} />}
        </Section>
      )}

      {/* RoPE */}
      {hasRope && (
        <Section title={t('modelDetail.sections.rope', 'RoPE 配置')}>
          {meta.ropeDim ? <DetailRow label={t('modelDetail.fields.ropeDim', '维度')} value={meta.ropeDim} /> : null}
          {meta.ropeFreqBase ? <DetailRow label={t('modelDetail.fields.ropeFreqBase', '频率基数')} value={meta.ropeFreqBase.toLocaleString()} /> : null}
          {meta.ropeFreqScale ? <DetailRow label={t('modelDetail.fields.ropeFreqScale', '频率缩放')} value={meta.ropeFreqScale} /> : null}
        </Section>
      )}

      {/* File Info */}
      <Section title={t('modelDetail.sections.fileInfo', '文件信息')}>
        {meta.fileSize ? <DetailRow label={t('modelDetail.fields.fileSize', '文件大小')} value={formatBytes(meta.fileSize)} /> : null}
        {meta.modelSize ? <DetailRow label={t('modelDetail.fields.modelSize', '权重大小')} value={formatBytes(meta.modelSize)} /> : null}
        {meta.bitsPerWeight ? <DetailRow label={t('modelDetail.fields.bitsPerWeight', '每权重位数')} value={`${meta.bitsPerWeight.toFixed(2)} bpw`} /> : null}
        {meta.fileTypeDescriptor && <DetailRow label={t('modelDetail.fields.fileTypeDescriptor', '文件类型')} value={meta.fileTypeDescriptor} />}
        {meta.poolingType !== undefined && meta.poolingType !== 0 && <DetailRow label={t('modelDetail.fields.poolingType', '池化类型')} value={meta.poolingType} />}
      </Section>

      {/* Status */}
      <Section title={t('modelDetail.sections.status', '状态')}>
        <DetailRow label={t('modelDetail.fields.status', '状态')} value={statusText} />
        {model.port ? <DetailRow label={t('modelDetail.fields.port', '端口')} value={model.port} /> : null}
        {model.pluginId && <DetailRow label={t('modelDetail.fields.pluginId', '后端')} value={model.pluginId} />}
        <DetailRow label={t('modelDetail.fields.scannedAt', '扫描时间')} value={new Date(model.scannedAt).toLocaleString('zh-CN')} />
      </Section>

      {/* Tags */}
      {model.tags && model.tags.length > 0 && (
        <Section title={t('modelDetail.sections.tags', '标签')}>
          <div className="flex flex-wrap gap-2 px-3 py-2">
            {model.tags.map((tag, i) => (
              <span key={i} className="px-2 py-0.5 text-xs font-medium bg-muted text-muted-foreground rounded-md">
                {tag}
              </span>
            ))}
          </div>
        </Section>
      )}
    </div>
  );
}
