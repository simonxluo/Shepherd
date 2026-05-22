import { useState, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Image, Loader2, Download, Copy, ImageOff } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { useLoadedModels, useAvailableModels } from '@/features/creative/hooks';
import { ModelSelect } from '@/features/creative/ModelSelect';
import { AvailableModelList } from '@/features/creative/AvailableModelList';
import { useImageGeneration } from '@/features/image-gen/hooks';
import { toast } from '@/hooks/useToast';

export function ImageGenPage() {
  const { t } = useTranslation();
  const { data: allModels = [] } = useLoadedModels();
  const imageModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.imageGeneration),
    [allModels]
  );
  const availableModels = useAvailableModels('imageGeneration');

  const imageGen = useImageGeneration();

  const [model, setModel] = useState('');
  const [prompt, setPrompt] = useState('');
  const [negativePrompt, setNegativePrompt] = useState('');
  const [size, setSize] = useState('1024x1024');
  const [n, setN] = useState(1);
  const [quality, setQuality] = useState('standard');
  const [style, setStyle] = useState('vivid');
  const [lastPrompt, setLastPrompt] = useState('');
  const downloadCounter = useRef(0);

  const handleGenerate = () => {
    if (!model) {
      toast.warning(t('imageGen.selectModelWarning', '请选择模型'));
      return;
    }
    if (!prompt.trim()) {
      toast.warning(t('imageGen.promptRequired', '请输入描述'));
      return;
    }

    setLastPrompt(prompt.trim());

    imageGen.mutate(
      {
        model,
        prompt: prompt.trim(),
        size: size || undefined,
        n: n > 1 ? n : undefined,
        quality: quality !== 'standard' ? quality : undefined,
        style: style !== 'vivid' ? style : undefined,
      },
      {
        onError: (error) => {
          toast.error(t('imageGen.generateFailed', '图像生成失败'), error.message);
        },
      }
    );
  };

  const handleDownload = (url: string) => {
    const a = document.createElement('a');
    a.href = url;
    a.download = `image_${++downloadCounter.current}.png`;
    a.target = '_blank';
    a.click();
  };

  const handleCopyPrompt = () => {
    if (lastPrompt) {
      navigator.clipboard.writeText(lastPrompt);
      toast.success(t('imageGen.promptCopied', '提示词已复制'));
    }
  };

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl mx-auto">
          {/* Header */}
          <div className="mb-6">
            <h1 className="text-2xl font-bold text-foreground">
              {t('imageGen.title', '图像生成')}
            </h1>
            <p className="text-sm text-muted-foreground mt-1">
              {t('imageGen.description', '通过 AI 模型从文本描述生成图像')}
            </p>
          </div>

          {imageModels.length === 0 ? (
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
                  models={imageModels}
                  value={model}
                  onValueChange={setModel}
                  placeholder={t('imageGen.selectModel', '选择模型')}
                  label={t('imageGen.modelLabel', '图像生成模型')}
                />

                <div>
                  <label className="block text-sm font-medium mb-1.5">
                    {t('imageGen.promptLabel', '图像描述')}
                  </label>
                  <Textarea
                    value={prompt}
                    onChange={(e) => setPrompt(e.target.value)}
                    placeholder={t('imageGen.promptPlaceholder', '描述要生成的图像...')}
                    className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
                    rows={3}
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1.5">
                    {t('imageGen.negativePromptLabel', '反向提示词 (可选)')}
                  </label>
                  <Textarea
                    value={negativePrompt}
                    onChange={(e) => setNegativePrompt(e.target.value)}
                    placeholder={t('imageGen.negativePromptPlaceholder', '描述要从图像中排除的内容...')}
                    className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
                    rows={2}
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('imageGen.sizeLabel', '尺寸')}
                    </label>
                    <Select value={size} onValueChange={setSize}>
                      <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="1024x1024">1024 × 1024</SelectItem>
                        <SelectItem value="1792x1024">1792 × 1024</SelectItem>
                        <SelectItem value="1024x1792">1024 × 1792</SelectItem>
                        <SelectItem value="512x512">512 × 512</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('imageGen.countLabel', '数量')}
                    </label>
                    <Select value={String(n)} onValueChange={(v) => setN(Number(v))}>
                      <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="1">1</SelectItem>
                        <SelectItem value="2">2</SelectItem>
                        <SelectItem value="4">4</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('imageGen.qualityLabel', '质量')}
                    </label>
                    <Select value={quality} onValueChange={setQuality}>
                      <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="standard">{t('imageGen.qualityStandard', '标准')}</SelectItem>
                        <SelectItem value="hd">{t('imageGen.qualityHD', '高清')}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('imageGen.styleLabel', '风格')}
                    </label>
                    <Select value={style} onValueChange={setStyle}>
                      <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="vivid">{t('imageGen.styleVivid', '生动')}</SelectItem>
                        <SelectItem value="natural">{t('imageGen.styleNatural', '自然')}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <Button
                  onClick={handleGenerate}
                  disabled={imageGen.isPending || !model || !prompt.trim()}
                  className="w-full"
                >
                  {imageGen.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      {t('imageGen.generating', '生成中...')}
                    </>
                  ) : (
                    <>
                      <Image className="w-4 h-4 mr-2" />
                      {t('imageGen.generate', '生成图像')}
                    </>
                  )}
                </Button>
              </div>

              {/* Results */}
              {imageGen.data && imageGen.data.data.length > 0 ? (
                <div className="border rounded-lg overflow-hidden">
                  {/* Result header */}
                  <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30">
                    <div className="flex items-center gap-2">
                      <h3 className="text-sm font-medium">
                        {t('imageGen.result', '生成结果')}
                      </h3>
                      <Badge variant="secondary" className="text-xs">
                        {t('imageGen.resultCount', '{{count}} 张图像', { count: imageGen.data.data.length })}
                      </Badge>
                    </div>
                    {lastPrompt && (
                      <Button
                        variant="ghost"
                        size="xs"
                        onClick={handleCopyPrompt}
                        className="text-xs text-muted-foreground hover:text-foreground"
                      >
                        <Copy className="w-3 h-3 mr-1" />
                        {t('imageGen.copyPrompt', '复制提示词')}
                      </Button>
                    )}
                  </div>

                  {/* Image grid */}
                  <div className="p-4">
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      {imageGen.data.data.map((item, index) => {
                        const src = item.b64_json
                          ? `data:image/png;base64,${item.b64_json}`
                          : item.url || '';

                        return (
                          <div key={index} className="rounded-lg overflow-hidden group relative border bg-muted/20">
                            {src && (
                              <>
                                <img
                                  src={src}
                                  alt={`${t('imageGen.imageAlt', '生成图像')} ${index + 1}`}
                                  className="w-full h-auto"
                                />
                                <div className="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors" />
                                <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                  <Button
                                    variant="secondary"
                                    size="xs"
                                    onClick={() => handleDownload(src)}
                                    className="shadow-md"
                                  >
                                    <Download className="w-3 h-3 mr-1" />
                                    {t('imageGen.download', '下载')}
                                  </Button>
                                </div>
                              </>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </div>
                </div>
              ) : !imageGen.isPending && !imageGen.data ? (
                /* Empty state */
                <div className="border rounded-lg p-8 text-center border-dashed">
                  <ImageOff className="w-12 h-12 mx-auto text-muted-foreground/50 mb-3" />
                  <p className="text-sm text-muted-foreground">
                    {t('imageGen.emptyState', '暂无生成的图像')}
                  </p>
                  <p className="text-xs text-muted-foreground/70 mt-1">
                    {t('imageGen.emptyStateHint', '输入描述并点击生成来创建图像')}
                  </p>
                </div>
              ) : null}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
