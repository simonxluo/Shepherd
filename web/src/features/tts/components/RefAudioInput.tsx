import { useState, useRef, useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Upload, Mic, MicOff, Link, Clock, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { toast } from '@/hooks/useToast';
import { useTTSHistory } from '../historyHooks';
import { getTTSAudioUrl, type TTSHistoryItem } from '../api';

type InputMode = 'upload' | 'record' | 'url' | 'history';

interface RefAudioInputProps {
  value: string;
  onChange: (base64OrUrl: string) => void;
}

export function RefAudioInput({ value, onChange }: RefAudioInputProps) {
  const { t } = useTranslation();
  const [mode, setMode] = useState<InputMode>('upload');
  const [recording, setRecording] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<Blob[]>([]);

  // 组件卸载时清理录音资源
  useEffect(() => {
    return () => {
      if (mediaRecorderRef.current && mediaRecorderRef.current.state !== 'inactive') {
        mediaRecorderRef.current.stop();
      }
      streamRef.current?.getTracks().forEach((t) => t.stop());
    };
  }, []);

  const handleFile = useCallback(async (file: File) => {
    const base64 = await audioFileToBase64(file);
    onChange(base64);
  }, [onChange]);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) handleFile(file);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    // 优先处理文件拖拽
    const file = e.dataTransfer.files[0];
    if (file && file.type.startsWith('audio/')) {
      handleFile(file);
      return;
    }
    // 处理 URL 拖拽（从 TTS 历史面板拖拽）
    const url = e.dataTransfer.getData('text/plain');
    if (url) {
      onChange(url);
    }
  };

  const toggleRecording = async () => {
    if (recording) {
      mediaRecorderRef.current?.stop();
      setRecording(false);
    } else {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        streamRef.current = stream;
        const recorder = new MediaRecorder(stream);
        chunksRef.current = [];

        recorder.ondataavailable = (e) => {
          if (e.data.size > 0) chunksRef.current.push(e.data);
        };

        recorder.onstop = async () => {
          stream.getTracks().forEach((t) => t.stop());
          streamRef.current = null;
          const blob = new Blob(chunksRef.current, { type: 'audio/webm' });
          const file = new File([blob], 'recording.webm', { type: 'audio/webm' });
          await handleFile(file);
        };

        recorder.start();
        mediaRecorderRef.current = recorder;
        setRecording(true);
      } catch {
        toast.error(t('tts.microphonePermissionDenied', '麦克风权限被拒绝，请在浏览器设置中允许访问'));
      }
    }
  };

  const handleSelectFromHistory = (item: TTSHistoryItem) => {
    const audioUrl = getTTSAudioUrl(item.id);
    onChange(audioUrl);
  };

  const clearValue = () => onChange('');

  const hasValue = !!value;

  return (
    <div className="space-y-2">
      {/* Mode switcher */}
      <div className="flex gap-1">
        {([
          { key: 'upload' as InputMode, icon: Upload, label: t('tts.refAudioUpload', '上传') },
          { key: 'record' as InputMode, icon: Mic, label: t('tts.refAudioRecord', '录制') },
          { key: 'url' as InputMode, icon: Link, label: t('tts.refAudioUrl', '链接') },
          { key: 'history' as InputMode, icon: Clock, label: t('tts.refAudioHistory', '历史') },
        ]).map(({ key, icon: Icon, label }) => (
          <Button
            key={key}
            variant={mode === key ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setMode(key)}
            className="text-xs"
          >
            <Icon className="w-3 h-3 mr-1" />
            {label}
          </Button>
        ))}
      </div>

      {/* Upload mode */}
      {mode === 'upload' && (
        <div
          onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
          onDragLeave={() => setDragOver(false)}
          onDrop={handleDrop}
          onClick={() => fileInputRef.current?.click()}
          className={`border-2 border-dashed rounded-lg p-4 text-center cursor-pointer transition-colors ${
            dragOver ? 'border-primary bg-primary/5' : 'border-muted-foreground/25 hover:border-primary/50'
          }`}
        >
          {hasValue ? (
            <div className="flex items-center justify-center gap-2">
              <span className="text-sm text-muted-foreground">{t('tts.refAudioLoaded', '已加载音频')}</span>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={(e) => { e.stopPropagation(); clearValue(); }}>
                <X className="w-3 h-3" />
              </Button>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">{t('tts.refAudioDragOrClick', '拖拽或点击上传音频文件')}</p>
          )}
          <input
            ref={fileInputRef}
            type="file"
            accept="audio/*"
            onChange={handleFileChange}
            className="hidden"
          />
        </div>
      )}

      {/* Record mode */}
      {mode === 'record' && (
        <div className="flex items-center gap-3">
          <Button
            variant={recording ? 'destructive' : 'outline'}
            size="sm"
            onClick={toggleRecording}
          >
            {recording ? <MicOff className="w-4 h-4 mr-1" /> : <Mic className="w-4 h-4 mr-1" />}
            {recording ? t('tts.stopRecord', '停止录制') : t('tts.startRecord', '开始录制')}
          </Button>
          {recording && (
            <span className="text-sm text-red-500 animate-pulse">{t('tts.refAudioRecording', '录制中...')}</span>
          )}
          {hasValue && !recording && (
            <span className="text-sm text-muted-foreground">{t('tts.refAudioRecorded', '已录制')}</span>
          )}
        </div>
      )}

      {/* URL mode */}
      {mode === 'url' && (
        <Input
          value={value && !value.startsWith('data:') ? value : ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={t('tts.refAudioUrlPlaceholder', '输入音频文件 URL...')}
          className="bg-background"
        />
      )}

      {/* History mode */}
      {mode === 'history' && (
        <HistoryPicker onSelect={handleSelectFromHistory} selectedUrl={value} />
      )}
    </div>
  );
}

/** Sub-component: pick a past TTS generation as reference */
function HistoryPicker({ onSelect, selectedUrl }: { onSelect: (item: TTSHistoryItem) => void; selectedUrl: string }) {
  const { t } = useTranslation();
  const { data, isLoading } = useTTSHistory({ limit: 20 });

  const items = data?.items ?? [];

  if (isLoading) {
    return <p className="text-xs text-muted-foreground py-2">{t('tts.historyLoading', 'Loading...')}</p>;
  }

  if (items.length === 0) {
    return <p className="text-xs text-muted-foreground py-2">{t('tts.noHistory', 'No TTS history yet')}</p>;
  }

  return (
    <div className="max-h-40 overflow-y-auto space-y-1 border rounded-lg p-2">
      {items.map((item) => {
        const itemUrl = getTTSAudioUrl(item.id);
        const isSelected = selectedUrl === itemUrl;
        return (
          <button
            key={item.id}
            onClick={() => onSelect(item)}
            className={`w-full text-left px-2 py-1.5 rounded text-sm hover:bg-accent/50 transition-colors ${
              isSelected ? 'bg-primary/10 border border-primary/30' : ''
            }`}
          >
            <p className="truncate">{item.inputText}</p>
            <p className="text-xs text-muted-foreground">{item.model} &middot; {item.format}</p>
          </button>
        );
      })}
    </div>
  );
}

async function audioFileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}
