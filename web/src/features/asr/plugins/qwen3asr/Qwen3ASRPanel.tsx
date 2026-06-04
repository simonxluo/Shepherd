import { useState, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Mic, Loader2, Upload, FileAudio, Copy, Check, AlertCircle, Play } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { ModelSelect } from '@/features/creative/ModelSelect';
import { AvailableModelList } from '@/features/creative/AvailableModelList';
import { useAvailableModels, BACKEND_LABELS } from '@/features/creative/hooks';
import { useLoadModel } from '@/features/models';
import { LoadModelDialog } from '@/features/models/components/LoadModelDialog';
import { toast } from '@/hooks/useToast';
import { formatBytes } from '@/lib/utils';
import { cn } from '@/lib/utils';
import type { ASRPluginPanelProps } from '../../types';

// Qwen3-ASR supported languages (30 languages + 22 Chinese dialects)
const QWEN3_ASR_LANGUAGES = [
  { value: '', label: 'asr.qwen3.langAuto', fallback: 'Auto Detect' },
  // Major languages
  { value: 'zh', label: 'Chinese (中文)' },
  { value: 'en', label: 'English' },
  { value: 'yue', label: 'Cantonese (粤语)' },
  { value: 'ja', label: 'Japanese (日本語)' },
  { value: 'ko', label: 'Korean (한국어)' },
  { value: 'de', label: 'German (Deutsch)' },
  { value: 'fr', label: 'French (Français)' },
  { value: 'es', label: 'Spanish (Español)' },
  { value: 'pt', label: 'Portuguese (Português)' },
  { value: 'ru', label: 'Russian (Русский)' },
  { value: 'it', label: 'Italian (Italiano)' },
  { value: 'ar', label: 'Arabic (العربية)' },
  { value: 'id', label: 'Indonesian (Bahasa Indonesia)' },
  { value: 'th', label: 'Thai (ภาษาไทย)' },
  { value: 'vi', label: 'Vietnamese (Tiếng Việt)' },
  { value: 'tr', label: 'Turkish (Türkçe)' },
  { value: 'hi', label: 'Hindi (हिन्दी)' },
  { value: 'ms', label: 'Malay (Bahasa Melayu)' },
  { value: 'nl', label: 'Dutch (Nederlands)' },
  { value: 'sv', label: 'Swedish (Svenska)' },
  { value: 'da', label: 'Danish (Dansk)' },
  { value: 'fi', label: 'Finnish (Suomi)' },
  { value: 'pl', label: 'Polish (Polski)' },
  { value: 'cs', label: 'Czech (Čeština)' },
  { value: 'fil', label: 'Filipino' },
  { value: 'fa', label: 'Persian (فارسی)' },
  { value: 'el', label: 'Greek (Ελληνικά)' },
  { value: 'hu', label: 'Hungarian (Magyar)' },
  { value: 'mk', label: 'Macedonian (Македонски)' },
  { value: 'ro', label: 'Romanian (Română)' },
];

export function Qwen3ASRPanel({
  model: selectedModel,
  matchedModels,
  onTranscribe,
  isTranscribing,
  result,
  onModelChange,
  modelStatus,
  fullModelId,
}: ASRPluginPanelProps) {
  const { t } = useTranslation();
  const availableModels = useAvailableModels('asr');
  const loadModel = useLoadModel();
  const [showLoadDialog, setShowLoadDialog] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Filter available models to only show Qwen3-ASR-related ones
  const qwen3AvailableModels = useMemo(
    () => availableModels.filter((m) => {
      const nameLower = (m.name || m.id || '').toLowerCase();
      return nameLower.includes('qwen3-asr') || nameLower.includes('qwen3asr') || nameLower.includes('qwen3_asr');
    }),
    [availableModels]
  );

  const modelName = selectedModel ? (selectedModel.alias || selectedModel.name) : '';

  // Model status
  const isModelRunning = !modelStatus || modelStatus === 'running';
  const isModelLoading = modelStatus === 'loading' || modelStatus === 'unloading';
  const isModelStopped = modelStatus === 'stopped';
  const isModelError = modelStatus === 'error';

  const [file, setFile] = useState<File | null>(null);
  const [language, setLanguage] = useState('');
  const [prompt, setPrompt] = useState('');
  const [responseFormat, setResponseFormat] = useState('text');
  const [temperature, setTemperature] = useState(0);
  const [copied, setCopied] = useState(false);

  const backendLabel = selectedModel?.pluginId
    ? BACKEND_LABELS[selectedModel.pluginId] || selectedModel.pluginId
    : '';

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
    if (isModelStopped) {
      setShowLoadDialog(true);
      return;
    }
    if (isModelLoading) {
      toast.info(t('asr.qwen3.modelLoading', 'Model is loading, please wait...'));
      return;
    }
    if (isModelError) {
      toast.error(t('asr.qwen3.modelError', 'Model encountered an error'));
      return;
    }
    if (!file) {
      toast.warning(t('asr.fileRequired', 'Please upload audio'));
      return;
    }

    onTranscribe({
      model: modelName,
      file,
      language: (language && language !== '_auto') ? language : undefined,
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
      <div className="space-y-4">
        <div className="flex flex-col items-center gap-3 py-6 px-4 rounded-lg border border-dashed border-orange-300 bg-orange-50/50 dark:border-orange-700 dark:bg-orange-950/20">
          <AlertCircle className="w-8 h-8 text-orange-500" />
          <div className="text-center">
            <p className="text-sm font-medium text-foreground">
              {t('asr.qwen3.notDetected', 'No loaded Qwen3-ASR model detected')}
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              {t('asr.qwen3.notDetectedHint', 'Please select a Qwen3-ASR model from the list below to load')}
            </p>
          </div>
        </div>
        <AvailableModelList
          models={qwen3AvailableModels}
          emptyText={t('asr.qwen3.noModels', 'No Qwen3-ASR models found')}
          emptyHint={t('asr.qwen3.noModelsHint', 'Please ensure Qwen3-ASR model paths are configured and scanned')}
        />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Model selection */}
      <div>
        <ModelSelect
          models={matchedModels}
          value={modelName}
          onValueChange={onModelChange}
          placeholder={t('asr.selectModel', 'Select ASR model')}
          label={t('asr.modelLabel', 'ASR Model')}
          showBackend
        />
        {backendLabel && (
          <p className="text-xs text-muted-foreground mt-1">
            {t('asr.qwen3.backend', 'Backend')}: {backendLabel}
          </p>
        )}
        {/* Model status indicator */}
        {modelName && !isModelRunning && (
          <div className={cn(
            'flex items-center gap-2 mt-2 px-3 py-2 rounded-md text-sm',
            isModelLoading && 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400',
            isModelStopped && 'bg-orange-500/10 text-orange-600 dark:text-orange-400',
            isModelError && 'bg-red-500/10 text-red-600 dark:text-red-400',
          )}>
            {isModelLoading && <Loader2 className="w-4 h-4 animate-spin" />}
            {(isModelStopped || isModelError) && <AlertCircle className="w-4 h-4" />}
            <span>
              {isModelLoading && t('asr.qwen3.modelLoading', 'Model is loading, please wait...')}
              {isModelStopped && t('asr.qwen3.modelNotLoaded', 'Model not loaded')}
              {isModelError && t('asr.qwen3.modelError', 'Model encountered an error')}
            </span>
            {isModelStopped && fullModelId && (
              <Button
                variant="outline"
                size="sm"
                className="ml-auto h-7 text-xs gap-1"
                onClick={() => setShowLoadDialog(true)}
              >
                <Play className="w-3 h-3" />
                {t('asr.qwen3.loadModel', 'Load Model')}
              </Button>
            )}
          </div>
        )}
      </div>

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

      {/* Language selector (Qwen3-ASR specific: dropdown with 52 languages) */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('asr.qwen3.languageLabel', 'Language')}
          </label>
          <Select value={language} onValueChange={setLanguage}>
            <SelectTrigger className="w-full bg-background">
              <SelectValue placeholder={t('asr.qwen3.langAuto', 'Auto Detect')} />
            </SelectTrigger>
            <SelectContent>
              {QWEN3_ASR_LANGUAGES.map((lang) => (
                <SelectItem key={lang.value || '_auto'} value={lang.value || '_auto'}>
                  {lang.fallback || lang.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-xs text-muted-foreground mt-1">
            {t('asr.qwen3.languageHint', 'Supports 30 languages + 22 Chinese dialects')}
          </p>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('asr.promptLabel', 'Prompt (optional)')}
          </label>
          <Input
            type="text"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
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
          <Select value={responseFormat} onValueChange={setResponseFormat}>
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
            onValueChange={([val]) => setTemperature(val)}
            min={0}
            max={1}
            step={0.1}
            className="w-full mt-2"
          />
        </div>
      </div>

      <Button
        onClick={handleTranscribe}
        disabled={isTranscribing || !modelName || !file || isModelLoading || isModelError}
        className="w-full"
      >
        {isTranscribing ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            {t('asr.transcribing', 'Transcribing...')}
          </>
        ) : isModelLoading ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            {t('asr.qwen3.modelLoading', 'Model is loading...')}
          </>
        ) : isModelStopped ? (
          <>
            <Mic className="w-4 h-4 mr-2" />
            {t('asr.qwen3.loadModelToTranscribe', 'Load Model & Transcribe')}
          </>
        ) : isModelError ? (
          <>
            <AlertCircle className="w-4 h-4 mr-2" />
            {t('asr.qwen3.modelError', 'Model error')}
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

      {/* Load model dialog */}
      {showLoadDialog && fullModelId && (
        <LoadModelDialog
          modelId={fullModelId}
          modelName={modelName}
          pluginId={selectedModel?.pluginId}
          isOpen={showLoadDialog}
          onClose={() => setShowLoadDialog(false)}
          onConfirm={(params) => {
            loadModel.mutate({ modelId: fullModelId, ...params });
            setShowLoadDialog(false);
          }}
        />
      )}
    </div>
  );
}
