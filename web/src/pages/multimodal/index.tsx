import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Wand2 } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { useLoadedModels } from '@/features/multimodal/hooks';
import { TTSPanel } from '@/features/multimodal/components/TTSPanel';
import { ASRPanel } from '@/features/multimodal/components/ASRPanel';
import { ImageGenPanel } from '@/features/multimodal/components/ImageGenPanel';

export function MultimodalPage() {
  const { t } = useTranslation();
  const { data: allModels = [] } = useLoadedModels();

  const ttsModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.tts),
    [allModels]
  );

  const asrModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.asr),
    [allModels]
  );

  const imageModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.imageGeneration),
    [allModels]
  );

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30">
        <div className="flex items-center gap-3">
          <Wand2 className="w-5 h-5 text-primary" />
          <h1 className="text-lg font-semibold">{t('multimodal.title', '多模态工具')}</h1>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl mx-auto">
          <Tabs defaultValue="tts">
            <TabsList className="w-full grid grid-cols-3">
              <TabsTrigger value="tts">
                {t('multimodal.tts', '语音合成 (TTS)')}
                {ttsModels.length > 0 && (
                  <span className="ml-1.5 text-xs text-muted-foreground">({ttsModels.length})</span>
                )}
              </TabsTrigger>
              <TabsTrigger value="asr">
                {t('multimodal.asr', '语音识别 (ASR)')}
                {asrModels.length > 0 && (
                  <span className="ml-1.5 text-xs text-muted-foreground">({asrModels.length})</span>
                )}
              </TabsTrigger>
              <TabsTrigger value="image">
                {t('multimodal.imageGen', '图像生成')}
                {imageModels.length > 0 && (
                  <span className="ml-1.5 text-xs text-muted-foreground">({imageModels.length})</span>
                )}
              </TabsTrigger>
            </TabsList>

            <TabsContent value="tts">
              <div className="mt-4">
                {ttsModels.length === 0 ? (
                  <div className="text-center py-12 text-muted-foreground">
                    <p>没有已加载的 TTS 模型</p>
                    <p className="text-sm mt-1">请先加载一个支持语音合成的模型</p>
                  </div>
                ) : (
                  <TTSPanel models={ttsModels} />
                )}
              </div>
            </TabsContent>

            <TabsContent value="asr">
              <div className="mt-4">
                {asrModels.length === 0 ? (
                  <div className="text-center py-12 text-muted-foreground">
                    <p>没有已加载的 ASR 模型</p>
                    <p className="text-sm mt-1">请先加载一个支持语音识别的模型</p>
                  </div>
                ) : (
                  <ASRPanel models={asrModels} />
                )}
              </div>
            </TabsContent>

            <TabsContent value="image">
              <div className="mt-4">
                {imageModels.length === 0 ? (
                  <div className="text-center py-12 text-muted-foreground">
                    <p>没有已加载的图像生成模型</p>
                    <p className="text-sm mt-1">请先加载一个支持图像生成的模型</p>
                  </div>
                ) : (
                  <ImageGenPanel models={imageModels} />
                )}
              </div>
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </div>
  );
}
