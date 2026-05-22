import { useState, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Mic, Loader2, Upload, FileAudio, Copy, Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { useLoadedModels, useAvailableModels } from '@/features/creative/hooks';
import { ModelSelect } from '@/features/creative/ModelSelect';
import { AvailableModelList } from '@/features/creative/AvailableModelList';
import { useASR } from '@/features/asr/hooks';
import { toast } from '@/hooks/useToast';
import { formatBytes } from '@/lib/utils';

export function ASRPage() {
  const { t } = useTranslation();
  const { data: allModels = [] } = useLoadedModels();
  const asrModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.asr),
    [allModels]
  );
  const availableModels = useAvailableModels('asr');

  const asr = useASR();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [model, setModel] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [language, setLanguage] = useState('');
  const [prompt, setPrompt] = useState('');
  const [responseFormat, setResponseFormat] = useState('text');
  const [temperature, setTemperature] = useState(0);
  const [copied, setCopied] = useState(false);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0];
    if (selected) {
      setFile(selected);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    const dropped = e.dataTransfer.files[0];
    if (dropped) {
      setFile(dropped);
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleTranscribe = () => {
    if (!model) {
      toast.warning(t('asr.selectModelWarning', '请选择模型'));
      return;
    }
    if (!file) {
      toast.warning(t('asr.fileRequired', '请上传音频'));
      return;
    }

    asr.mutate(
      {
        model,
        file,
        language: language || undefined,
        prompt: prompt || undefined,
        response_format: responseFormat !== 'text' ? responseFormat : undefined,
        temperature: temperature > 0 ? temperature : undefined,
      },
      {
        onError: (error) => {
          toast.error(t('asr.transcribeFailed', '语音识别失败'), error.message);
        },
      }
    );
  };

  const handleCopy = () => {
    if (asr.data?.text) {
      navigator.clipboard.writeText(asr.data.text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const resultText = asr.data?.text ?? '';

  const wordCount = useMemo(() => {
    if (!resultText) return 0;
    return resultText.trim().split(/\s+/).filter(Boolean).length;
  }, [resultText]);

  const charCount = useMemo(() => {
    return resultText.length;
  }, [resultText]);

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl mx-auto">
          {/* Header */}
          <div className="mb-6">
            <h1 className="text-2xl font-bold text-foreground">
              {t('asr.title', '语音识别 (ASR)')}
            </h1>
            <p className="text-sm text-muted-foreground mt-1">
              {t('asr.description', '上传音频文件进行语音识别转录')}
            </p>
          </div>

          {asrModels.length === 0 ? (
            <AvailableModelList
              models={availableModels}
              emptyText={t('creative.noScannedModels')}
              emptyHint={t('creative.noScannedModelsHint')}
            />
          ) : (
            <div className="space-y-6">
              {/* Form */}
              <div className="space-y-4">
                <ModelSelect
                  models={asrModels}
                  value={model}
                  onValueChange={setModel}
                  placeholder={t('asr.selectModel', '选择 ASR 模型')}
                  label={t('asr.modelLabel', 'ASR 模型')}
                />

                {/* File upload area */}
                <div>
                  <label className="block text-sm font-medium mb-1.5">
                    {t('asr.fileLabel', '音频文件')}
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
                          {t('asr.dropzone', '拖拽音频文件到此处，或点击上传')}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {t('asr.supportedFormats', '支持 WAV, MP3, OGG, FLAC, M4A, WebM')}
                        </p>
                      </div>
                    )}
                  </div>
                </div>

                {/* Language and Prompt */}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('asr.languageLabel', '语言 (可选)')}
                    </label>
                    <Input
                      type="text"
                      value={language}
                      onChange={(e) => setLanguage(e.target.value)}
                      placeholder={t('asr.languagePlaceholder', '如：zh, en, ja')}
                      className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('asr.promptLabel', '提示词 (可选)')}
                    </label>
                    <Input
                      type="text"
                      value={prompt}
                      onChange={(e) => setPrompt(e.target.value)}
                      placeholder={t('asr.promptPlaceholder', '可选的提示文本')}
                      className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent"
                    />
                  </div>
                </div>

                {/* Response format and Temperature */}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('asr.responseFormatLabel', '响应格式')}
                    </label>
                    <Select value={responseFormat} onValueChange={setResponseFormat}>
                      <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
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
                      {t('asr.temperatureLabel', '温度')}: {temperature.toFixed(1)}
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
                  disabled={asr.isPending || !model || !file}
                  className="w-full"
                >
                  {asr.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      {t('asr.transcribing', '识别中...')}
                    </>
                  ) : (
                    <>
                      <Mic className="w-4 h-4 mr-2" />
                      {t('asr.transcribe', '开始识别')}
                    </>
                  )}
                </Button>
              </div>

              {/* Result */}
              {asr.data && (
                <div className="border rounded-lg overflow-hidden">
                  {/* Result header */}
                  <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30">
                    <h3 className="text-sm font-medium">
                      {t('asr.result', '识别结果')}
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
                          {t('asr.copied', '已复制')}
                        </>
                      ) : (
                        <>
                          <Copy className="w-3 h-3 mr-1" />
                          {t('asr.copy', '复制')}
                        </>
                      )}
                    </Button>
                  </div>

                  {/* Result body */}
                  <div className="p-4 space-y-3">
                    <p className="text-sm whitespace-pre-wrap bg-muted/50 rounded-md p-3 leading-relaxed">
                      {asr.data.text}
                    </p>

                    {/* Metadata */}
                    <div className="flex flex-wrap gap-3 text-xs text-muted-foreground pt-1">
                      {asr.data.language && (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-muted">
                          {t('asr.detectedLanguage', '语言')}: {asr.data.language}
                        </span>
                      )}
                      {asr.data.duration && (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-muted">
                          {t('asr.duration', '时长')}: {asr.data.duration.toFixed(1)}s
                        </span>
                      )}
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-muted">
                        {t('asr.wordCount', '词数')}: {wordCount}
                      </span>
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-muted">
                        {t('asr.charCount', '字符数')}: {charCount}
                      </span>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
