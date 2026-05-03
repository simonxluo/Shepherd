import { useState, useRef } from 'react';
import { Image, Loader2, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useImageGeneration } from '../hooks';
import { useToast } from '@/hooks/useToast';

interface ImageGenPanelProps {
  models: Array<{ id: string; name: string; alias?: string }>;
}

export function ImageGenPanel({ models }: ImageGenPanelProps) {
  const toast = useToast();
  const imageGen = useImageGeneration();

  const [model, setModel] = useState('');
  const [prompt, setPrompt] = useState('');
  const [size, setSize] = useState('1024x1024');
  const [n, setN] = useState(1);
  const [quality, setQuality] = useState('standard');
  const [style, setStyle] = useState('vivid');
  const downloadCounter = useRef(0);

  const handleGenerate = () => {
    if (!model) {
      toast.warning('请选择模型', '请从下拉列表中选择一个支持图像生成的模型');
      return;
    }
    if (!prompt.trim()) {
      toast.warning('请输入描述', '请输入要生成的图像描述');
      return;
    }

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
          toast.error('图像生成失败', error.message);
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

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1.5">图像生成模型</label>
          <Select
            value={model}
            onValueChange={setModel}
          >
            <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
              <SelectValue placeholder="选择模型" />
            </SelectTrigger>
            <SelectContent>
              {models.map((m) => (
                <SelectItem key={m.id} value={m.alias || m.name}>
                  {m.alias || m.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1.5">图像描述</label>
          <Textarea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="描述要生成的图像..."
            className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
            rows={3}
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1.5">尺寸</label>
            <Select
              value={size}
              onValueChange={setSize}
            >
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
            <label className="block text-sm font-medium mb-1.5">数量</label>
            <Select
              value={String(n)}
              onValueChange={(v) => setN(Number(v))}
            >
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
            <label className="block text-sm font-medium mb-1.5">质量</label>
            <Select
              value={quality}
              onValueChange={setQuality}
            >
              <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="standard">标准</SelectItem>
                <SelectItem value="hd">高清</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1.5">风格</label>
            <Select
              value={style}
              onValueChange={setStyle}
            >
              <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="vivid">生动</SelectItem>
                <SelectItem value="natural">自然</SelectItem>
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
              生成中...
            </>
          ) : (
            <>
              <Image className="w-4 h-4 mr-2" />
              生成图像
            </>
          )}
        </Button>
      </div>

      {imageGen.data && imageGen.data.data.length > 0 && (
        <div className="space-y-4">
          <h3 className="text-sm font-medium">生成结果</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {imageGen.data.data.map((item, index) => {
              const src = item.b64_json
                ? `data:image/png;base64,${item.b64_json}`
                : item.url || '';

              return (
                <div key={index} className="border rounded-lg overflow-hidden group relative">
                  {src && (
                    <>
                      <img
                        src={src}
                        alt={`生成图像 ${index + 1}`}
                        className="w-full h-auto"
                      />
                      <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button
                          variant="secondary"
                          size="xs"
                          onClick={() => handleDownload(src)}
                        >
                          <Download className="w-3 h-3" />
                        </Button>
                      </div>
                    </>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
