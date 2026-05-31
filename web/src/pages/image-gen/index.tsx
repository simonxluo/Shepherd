import { useState, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Image, Loader2, Download, Copy, ImageOff } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { useLoadedModels, useAvailableModels } from '@/features/creative/hooks';
import { ModelSelect } from '@/components/model/ModelSelect';
import { AvailableModelList } from '@/components/model/AvailableModelList';
import { useImageGeneration } from '@/features/image-gen/hooks';
import { useImageGenStore } from '@/stores/imageGenStore';
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

  // 表单状态从 Zustand store 获取（跨页面持久化）
  const form = useImageGenStore((s) => s.form);
  const setField = useImageGenStore((s) => s.setField);

  // 瞬态 UI 状态保留为本地 useState
  const [lastPrompt, setLastPrompt] = useState('');
  const downloadCounter = useRef(0);

  const handleGenerate = () => {
    const currentForm = useImageGenStore.getState().form;
    if (!currentForm.model) {
      toast.warning(t('imageGen.selectModelWarning', '请选择模型'));
      return;
    }
    if (!currentForm.prompt.trim()) {
      toast.warning(t('imageGen.promptRequired', '请输入描述'));
      return;
    }

    setLastPrompt(currentForm.prompt.trim());

    imageGen.mutate(
      {
        model: currentForm.model,
        prompt: currentForm.prompt.trim(),
        size: currentForm.size || undefined,
        n: currentForm.n > 1 ? currentForm.n : undefined,
        quality: currentForm.quality !== 'standard' ? currentForm.quality : undefined,
        style: currentForm.style !== 'vivid' ? currentForm.style : undefined,
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
    document.body.appendChild(a);
    a.click();
    setTimeout(() => document.body.removeChild(a), 100);
  };

  const handleCopyPrompt = () => {
    if (lastPrompt) {
      navigator.clipboard.writeText(lastPrompt).then(
        () => toast.success(t('imageGen.promptCopied', '提示词已复制')),
        () => toast.error(t('imageGen.copyFailed', '复制失败，请手动复制'))
      );
    }
  };

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      <div className="flex-1 overflow-y-auto p-6">
        {/* Header — left-aligned at the top */}
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-foreground">
            {t('imageGen.title', '图像生成')}
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t('imageGen.description', '通过 AI 模型从文本描述生成图像')}
          </p>
        </div>

        {/* Operation UI — centered */}
        <div className="max-w-3xl mx-auto">
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
                  value={form.model}
                  onValueChange={(v) => setField('model', v)}
                  placeholder={t('imageGen.selectModel', '选择模型')}
                  label={t('imageGen.modelLabel', '图像生成模型')}
                />

                <div>
                  <label className="block text-sm font-medium mb-1.5">
                    {t('imageGen.promptLabel', '图像描述')}
                  </label>
                  <Textarea
                    value={form.prompt}
                    onChange={(e) => setField('prompt', e.target.value)}
                    placeholder={t('imageGen.promptPlaceholder', '描述要生成的图像...')}
                    className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
                    rows={3}
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('imageGen.sizeLabel', '尺寸')}
                    </label>
                    <Select value={form.size} onValueChange={(v) => setField('size', v)}>
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
                    <Select value={String(form.n)} onValueChange={(v) => setField('n', Number(v))}>
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
                    <Select value={form.quality} onValueChange={(v) => setField('quality', v)}>
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
                    <Select value={form.style} onValueChange={(v) => setField('style', v)}>
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
                  disabled={imageGen.isPending || !form.model || !form.prompt.trim()}
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
