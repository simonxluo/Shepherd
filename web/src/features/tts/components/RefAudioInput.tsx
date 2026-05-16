import { useState, useRef, useCallback } from 'react';
import { Upload, Mic, MicOff, Link, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

type InputMode = 'upload' | 'record' | 'url';

interface RefAudioInputProps {
  value: string;
  onChange: (base64OrUrl: string) => void;
}

export function RefAudioInput({ value, onChange }: RefAudioInputProps) {
  const [mode, setMode] = useState<InputMode>('upload');
  const [recording, setRecording] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<Blob[]>([]);

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
    const file = e.dataTransfer.files[0];
    if (file && file.type.startsWith('audio/')) handleFile(file);
  };

  const toggleRecording = async () => {
    if (recording) {
      mediaRecorderRef.current?.stop();
      setRecording(false);
    } else {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        const recorder = new MediaRecorder(stream);
        chunksRef.current = [];

        recorder.ondataavailable = (e) => {
          if (e.data.size > 0) chunksRef.current.push(e.data);
        };

        recorder.onstop = async () => {
          stream.getTracks().forEach((t) => t.stop());
          const blob = new Blob(chunksRef.current, { type: 'audio/webm' });
          const file = new File([blob], 'recording.webm', { type: 'audio/webm' });
          await handleFile(file);
        };

        recorder.start();
        mediaRecorderRef.current = recorder;
        setRecording(true);
      } catch {
        // 麦克风权限被拒绝
      }
    }
  };

  const clearValue = () => onChange('');

  const hasValue = !!value;

  return (
    <div className="space-y-2">
      {/* 模式切换 */}
      <div className="flex gap-1">
        {([
          { key: 'upload' as InputMode, icon: Upload, label: '上传' },
          { key: 'record' as InputMode, icon: Mic, label: '录制' },
          { key: 'url' as InputMode, icon: Link, label: '链接' },
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

      {/* 上传模式 */}
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
              <span className="text-sm text-muted-foreground">已加载音频</span>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={(e) => { e.stopPropagation(); clearValue(); }}>
                <X className="w-3 h-3" />
              </Button>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">拖拽或点击上传音频文件</p>
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

      {/* 录制模式 */}
      {mode === 'record' && (
        <div className="flex items-center gap-3">
          <Button
            variant={recording ? 'destructive' : 'outline'}
            size="sm"
            onClick={toggleRecording}
          >
            {recording ? <MicOff className="w-4 h-4 mr-1" /> : <Mic className="w-4 h-4 mr-1" />}
            {recording ? '停止录制' : '开始录制'}
          </Button>
          {recording && (
            <span className="text-sm text-red-500 animate-pulse">录制中...</span>
          )}
          {hasValue && !recording && (
            <span className="text-sm text-muted-foreground">已录制</span>
          )}
        </div>
      )}

      {/* URL 模式 */}
      {mode === 'url' && (
        <Input
          value={value && !value.startsWith('data:') ? value : ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder="输入音频文件 URL..."
          className="bg-background"
        />
      )}
    </div>
  );
}

async function audioFileToBase64(file: File): Promise<string> {
  const buffer = await file.arrayBuffer();
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return `data:${file.type};base64,${btoa(binary)}`;
}
