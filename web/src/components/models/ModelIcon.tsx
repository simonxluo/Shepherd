import { type ComponentType } from 'react';
import { Brain } from 'lucide-react';

interface ModelIconProps {
  architecture: string;
  className?: string;
}

/**
 * 模型图标组件 - 支持从底部到顶部的发光动画
 * 正常状态静态，悬停时播放发光动画
 */
export function ModelIcon({ architecture, className = '' }: ModelIconProps) {
  const arch = architecture.toLowerCase();
  let colorClass = 'text-muted-foreground';

  // 设置颜色
  if (arch.includes('qwen')) colorClass = 'text-blue-500';
  else if (arch.includes('llama')) colorClass = 'text-amber-500';
  else if (arch.includes('mistral')) colorClass = 'text-purple-500';
  else if (arch.includes('gemma')) colorClass = 'text-pink-500';
  else if (arch.includes('deepseek')) colorClass = 'text-cyan-500';

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      className={`model-icon-glow ${colorClass} ${className}`}
      style={{ overflow: 'visible' }}
    >
      {/* Brain icon paths */}
      <path
        d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 1.98-3A2.5 2.5 0 0 1 9.5 2Z"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="brain-path-left"
      />
      <path
        d="M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-1.98-3A2.5 2.5 0 0 0 14.5 2Z"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="brain-path-right"
      />
      <style>{`
        /* 默认状态：静态显示，无动画 */
        .model-icon-glow .brain-path-left,
        .model-icon-glow .brain-path-right {
          stroke-dasharray: 100;
          stroke-dashoffset: 0;
          transition: all 0.3s ease;
        }

        /* 悬停时：播放发光动画 */
        .model-icon-glow:hover .brain-path-left {
          animation: glowStrokeLeft 2s ease-in-out infinite;
        }

        .model-icon-glow:hover .brain-path-right {
          animation: glowStrokeRight 2s ease-in-out infinite;
          animation-delay: 0.4s;
        }

        /* 左脑路径动画 */
        @keyframes glowStrokeLeft {
          0% {
            stroke-dashoffset: 100;
            filter: drop-shadow(0 0 0px currentColor);
            opacity: 0.3;
          }
          25% {
            stroke-dashoffset: 0;
            opacity: 1;
          }
          50% {
            stroke-dashoffset: 0;
            filter: drop-shadow(0 0 6px currentColor) drop-shadow(0 0 10px currentColor) drop-shadow(0 0 14px currentColor);
            opacity: 1;
          }
          75% {
            stroke-dashoffset: 0;
            opacity: 1;
          }
          100% {
            stroke-dashoffset: -100;
            filter: drop-shadow(0 0 0px currentColor);
            opacity: 0.3;
          }
        }

        /* 右脑路径动画 */
        @keyframes glowStrokeRight {
          0% {
            stroke-dashoffset: 100;
            filter: drop-shadow(0 0 0px currentColor);
            opacity: 0.3;
          }
          25% {
            stroke-dashoffset: 0;
            opacity: 1;
          }
          50% {
            stroke-dashoffset: 0;
            filter: drop-shadow(0 0 6px currentColor) drop-shadow(0 0 10px currentColor) drop-shadow(0 0 14px currentColor);
            opacity: 1;
          }
          75% {
            stroke-dashoffset: 0;
            opacity: 1;
          }
          100% {
            stroke-dashoffset: -100;
            filter: drop-shadow(0 0 0px currentColor);
            opacity: 0.3;
          }
        }
      `}</style>
    </svg>
  );
}
