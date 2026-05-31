import { useState, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Mic, Loader2, Upload, FileAudio, Copy, Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { ModelSelect } from '@/components/model/ModelSelect';
import { AvailableModelList } from '@/components/model/AvailableModelList';
import { useAvailableModels } from '@/features/creative/hooks';
import { useASRStore } from '@/stores/asrStore';
import { toast } from '@/hooks/useToast';
import { formatBytes } from '@/lib/utils';
import type { ASRPluginPanelProps } from '@/features/asr/types';

export function GenericASRPanel({
  model: selectedModel,
  matchedModels,
  onTranscribe,
  isTranscribing,
  result,
  onModelChange,
}: ASRPluginPanelProps) {
  const { t } = useTranslation();
  const availableModels = useAvailableModels('asr');
  const fileInputRef = useRef<HTMLInputElement>(null);

  const modelName = selectedModel ? (selectedModel.alias || selectedModel.name) : '';

  // 表单状态从 Zustand store 获取（跨页面持久化）
  const genericForm = useASRStore((s) => s.genericForm);
  const setGenericField = useASRStore((s) => s.setGenericField);
  const { language, prompt, responseFormat, temperature } = genericForm;

  // 瞬态 UI 状态保留为本地 useState
  const [file, setFile] = useState<File | null>(null);
  const [copied, setCopied] = useState(false);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0];
    if (selected) setFile(selected);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    const dropped = e.dataTransfer.files[0];
    if (dropped) setFile(dropped);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleTranscribe = () => {
    if (!modelName) {
      toast.warning(t('asr.selectModelWarning', 'Please select a model'));
      return;
    }
    if (!file) {
      toast.warning(t('asr.fileRequired', 'Please upload audio'));
      return;
    }

    onTranscribe({
      model: modelName,
      file,
      language: language || undefined,
      prompt: prompt || undefined,
      response_format: responseFormat !== 'text' ? responseFormat : undefined,
      temperature: temperature > 0 ? temperature : undefined,
    });
  };

  const handleCopy = () => {
    if (result?.text) {
      navigator.clipboard.writeText(result.text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const resultText = result?.text ?? '';

  const wordCount = useMemo(() => {
    if (!resultText) return 0;
    return resultText.trim().split(/\s+/).filter(Boolean).length;
  }, [resultText]);

  const charCount = useMemo(() => resultText.length, [resultText]);

  // Empty state
  if (matchedModels.length === 0) {
    return (
      <AvailableModelList
        models={availableModels}
        emptyText={t('creative.noScannedModels')}
        emptyHint={t('creative.noScannedModelsHint')}
      />
    );
  }

  return (
    <div className="space-y-4">
      {/* Model selection */}
      <ModelSelect
        models={matchedModels}
        value={modelName}
        onValueChange={onModelChange}
        placeholder={t('asr.selectModel', 'Select ASR model')}
        label={t('asr.modelLabel', 'ASR Model')}
      />

      {/* File upload area */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('asr.fileLabel', 'Audio File')}
        </label>
        <div
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onClick={() => fileInputRef.current?.click()}
          className="border-2 border-dashed rounded-lg p-6 text-center cursor-pointer hover:border-primary/50 hover:bg-accent/50 transition-colors"
        >
          <input
            ref={fileInputRef}
            type="file"
            accept="audio/*,.wav,.mp3,.ogg,.flac,.m4a,.webm"
            onChange={handleFileChange}
            className="hidden"
          />
          {file ? (
            <div className="flex items-center justify-center gap-2">
              <FileAudio className="w-5 h-5 text-primary" />
              <span className="text-sm">{file.name}</span>
              <span className="text-xs text-muted-foreground">({formatBytes(file.size)})</span>
            </div>
          ) : (
            <div className="space-y-2">
              <Upload className="w-8 h-8 mx-auto text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                {t('asr.dropzone', 'Drag and drop an audio file here, or click to upload')}
              </p>
              <p className="text-xs text-muted-foreground">
                {t('asr.supportedFormats', 'Supports WAV, MP3, OGG, FLAC, M4A, WebM')}
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Language and Prompt */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('asr.languageLabel', 'Language (optional)')}
          </label>
          <Input
            type="text"
            value={language}
            onChange={(e) => setGenericField('language', e.target.value)}
            placeholder={t('asr.languagePlaceholder', 'e.g., zh, en, ja')}
            className="w-full"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('asr.promptLabel', 'Prompt (optional)')}
          </label>
          <Input
            type="text"
            value={prompt}
            onChange={(e) => setGenericField('prompt', e.target.value)}
            placeholder={t('asr.promptPlaceholder', 'Optional prompt text')}
            className="w-full"
          />
        </div>
      </div>

      {/* Response format and Temperature */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('asr.responseFormatLabel', 'Response Format')}
          </label>
          <Select value={responseFormat} onValueChange={(v) => setGenericField('responseFormat', v)}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="text">text</SelectItem>
              <SelectItem value="json">json</SelectItem>
              <SelectItem value="verbose_json">verbose_json</SelectItem>
              <SelectItem value="srt">srt</SelectItem>
              <SelectItem value="vtt">vtt</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('asr.temperatureLabel', 'Temperature')}: {temperature.toFixed(1)}
          </label>
          <Slider
            value={[temperature]}
            onValueChange={([val]) => setGenericField('temperature', val)}
            min={0}
            max={1}
            step={0.1}
            className="w-full mt-2"
          />
        </div>
      </div>

      <Button
        onClick={handleTranscribe}
        disabled={isTranscribing || !modelName || !file}
        className="w-full"
      >
        {isTranscribing ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            {t('asr.transcribing', 'Transcribing...')}
          </>
        ) : (
          <>
            <Mic className="w-4 h-4 mr-2" />
            {t('asr.transcribe', 'Transcribe')}
          </>
        )}
      </Button>

      {/* Result */}
      {result && (
        <div className="border rounded-lg overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30">
            <h3 className="text-sm font-medium">
              {t('asr.result', 'Result')}
            </h3>
            <Button
              variant="ghost"
              size="xs"
              onClick={handleCopy}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              {copied ? (
                <>
                  <Check className="w-3 h-3 mr-1 text-green-500" />
                  {t('asr.copied', 'Copied')}
                </>
              ) : (
                <>
                  <Copy className="w-3 h-3 mr-1" />
                  {t('asr.copy', 'Copy')}
                </>
              )}
            </Button>
          </div>
          <div className="p-4 space-y-3">
            <p className="text-sm whitespace-pre-wrap bg-muted/50 rounded-md p-3 leading-relaxed">
              {result.text}
            </p>
            <div className="flex flex-wrap gap-3 text-xs text-muted-foreground pt-1">
              {result.language && (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-muted">
                  {t('asr.detectedLanguage', 'Language')}: {result.language}
                </span>
              )}
              {result.duration && (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-muted">
                  {t('asr.duration', 'Duration')}: {result.duration.toFixed(1)}s
                </span>
              )}
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-muted">
                {t('asr.wordCount', 'Words')}: {wordCount}
              </span>
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-muted">
                {t('asr.charCount', 'Characters')}: {charCount}
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
