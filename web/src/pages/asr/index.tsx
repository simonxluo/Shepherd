import { useState, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Mic, Loader2, Upload, FileAudio } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useLoadedModels } from '@/features/creative/hooks';
import { ModelSelect } from '@/features/creative/ModelSelect';
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

  const asr = useASR();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [model, setModel] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [language, setLanguage] = useState('');
  const [prompt, setPrompt] = useState('');

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
      },
      {
        onError: (error) => {
          toast.error(t('asr.transcribeFailed', '语音识别失败'), error.message);
        },
      }
    );
  };

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl mx-auto">
          <div className="mb-6">
            <h1 className="text-2xl font-bold text-foreground">
              {t('asr.title', '语音识别 (ASR)')}
            </h1>
          </div>

          {asrModels.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <p>{t('asr.noModels', '没有已加载的 ASR 模型')}</p>
              <p className="text-sm mt-1">{t('asr.noModelsHint', '请先加载一个支持语音识别的模型')}</p>
            </div>
          ) : (
            <div className="space-y-6">
              <div className="space-y-4">
                <ModelSelect
                  models={asrModels}
                  value={model}
                  onValueChange={setModel}
                  placeholder={t('asr.selectModel', '选择 ASR 模型')}
                  label={t('asr.modelLabel', 'ASR 模型')}
                />

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

              {asr.data && (
                <div className="border rounded-lg p-4 space-y-2">
                  <h3 className="text-sm font-medium">{t('asr.result', '识别结果')}</h3>
                  <p className="text-sm whitespace-pre-wrap bg-muted/50 rounded-md p-3">
                    {asr.data.text}
                  </p>
                  {(asr.data.language || asr.data.duration) && (
                    <div className="flex gap-4 text-xs text-muted-foreground">
                      {asr.data.language && <span>{t('asr.detectedLanguage', '语言')}: {asr.data.language}</span>}
                      {asr.data.duration && <span>{t('asr.duration', '时长')}: {asr.data.duration.toFixed(1)}s</span>}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
