interface ModelIconProps {
  architecture: string;
  className?: string;
}

const MODEL_ICONS: [string, string][] = [
  ['qwen', '/model-icons/qwen-color.svg'],
  ['llama', '/model-icons/meta-color.svg'],
  ['mistral', '/model-icons/mistral-color.svg'],
  ['gemma', '/model-icons/gemma-color.svg'],
  ['deepseek', '/model-icons/deepseek-color.svg'],
  ['glm', '/model-icons/glm-color.svg'],
  ['kimi', '/model-icons/kimi-color.svg'],
  ['minimax', '/model-icons/minimax-color.svg'],
  ['hunyuan', '/model-icons/hunyuan-color.svg'],
  ['nemotron', '/model-icons/nemotron-color.svg'],
  ['openai', '/model-icons/openai-color.svg'],
  ['bailingmoe', '/model-icons/bailingmoe-color.svg'],
  ['bert', '/model-icons/bert-color.svg'],
  ['lfm', '/model-icons/lfm-color.svg'],
  ['paddleocr', '/model-icons/paddleocr-color.svg'],
  ['seed_oss', '/model-icons/seed_oss-color.svg'],
  ['step35', '/model-icons/step35-color.svg'],
];

export function ModelIcon({ architecture, className = '' }: ModelIconProps) {
  const arch = architecture.toLowerCase();

  const iconSrc = MODEL_ICONS.find(([key]) => arch.includes(key))?.[1];

  if (iconSrc) {
    return (
      <img
        src={iconSrc}
        alt={architecture}
        className={`model-icon ${className}`}
      />
    );
  }

  // Default: Brain icon
  const path1 =
    'M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 1.98-3A2.5 2.5 0 0 1 9.5 2Z';
  const path2 =
    'M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-1.98-3A2.5 2.5 0 0 0 14.5 2Z';

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      className={`model-icon text-muted-foreground ${className}`}
      style={{ overflow: 'visible' }}
    >
      <path
        d={path1}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d={path2}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
