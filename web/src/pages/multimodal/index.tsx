import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
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
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl mx-auto">
          <div className="mb-6">
            <h1 className="text-2xl font-bold text-foreground">{t('multimodal.title', '多模态工具')}</h1>
          </div>

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
                    <p>{t('multimodal.noTTSModels')}</p>
                    <p className="text-sm mt-1">{t('multimodal.noTTSModelsHint')}</p>
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
                    <p>{t('multimodal.noASRModels')}</p>
                    <p className="text-sm mt-1">{t('multimodal.noASRModelsHint')}</p>
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
                    <p>{t('multimodal.noImageModels')}</p>
                    <p className="text-sm mt-1">{t('multimodal.noImageModelsHint')}</p>
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
