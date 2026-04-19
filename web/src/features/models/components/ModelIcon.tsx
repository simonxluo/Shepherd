interface ModelIconProps {
  architecture: string;
  className?: string;
}

/**
 * Model icon component with glow animation on hover
 */
export function ModelIcon({ architecture, className = '' }: ModelIconProps) {
  const arch = architecture.toLowerCase();
  let colorClass = 'text-muted-foreground';
  let path1 = '';
  let path2 = '';

  if (arch.includes('qwen')) {
    colorClass = 'text-blue-500';
    path1 = 'M 12 4 A 8 8 0 0 0 4 12 A 8 8 0 0 0 12 20 M 12 10 A 2 2 0 1 0 12 14 A 2 2 0 1 0 12 10';
    path2 = 'M 12 4 A 8 8 0 0 1 20 12 A 8 8 0 0 1 17.6 17.6 L 22 22';
  } else if (arch.includes('llama')) {
    colorClass = 'text-amber-500';
    path1 = 'M 12 5 V 2 L 11 4 L 10 2 V 9 H 7 V 12 H 10 V 20';
    path2 = 'M 12 5 V 12 H 17 V 20 M 17 14 H 19';
  } else if (arch.includes('mistral')) {
    colorClass = 'text-purple-500';
    path1 = 'M 6 21 C 2 21 2 17 4 11 C 5 7 9 5 12 11';
    path2 = 'M 12 11 C 15 5 19 7 20 11 C 22 17 22 21 18 21';
  } else if (arch.includes('gemma')) {
    colorClass = 'text-pink-500';
    path1 = 'M 12 2 L 21 9 L 12 22 L 3 9 Z';
    path2 = 'M 3 9 L 12 13 L 21 9 M 12 2 V 22';
  } else if (arch.includes('deepseek')) {
    colorClass = 'text-cyan-500';
    path1 = 'M 2 12 C 7 6 17 6 22 12 C 17 18 7 18 2 12 Z';
    path2 = 'M 12 15 A 3 3 0 1 0 12 9 A 3 3 0 1 0 12 15 M 12 7 V 9 M 12 15 V 17 M 7 12 H 9 M 15 12 H 17';
  } else {
    // Default (Brain)
    path1 = 'M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 1.98-3A2.5 2.5 0 0 1 9.5 2Z';
    path2 = 'M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-1.98-3A2.5 2.5 0 0 0 14.5 2Z';
  }
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
      {/* Dynamic icon paths */}
      <path
        d={path1}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="glow-path-1"
      />
      <path
        d={path2}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="glow-path-2"
      />
      <style>{`
        /* Default: static */
        .model-icon-glow .glow-path-1,
        .model-icon-glow .glow-path-2 {
          stroke-dasharray: 100;
          stroke-dashoffset: 0;
          transition: all 0.3s ease;
        }

        /* Hover: glow animation */
        .model-icon-glow:hover .glow-path-1 {
          animation: glowStroke1 2s ease-in-out infinite;
        }

        .model-icon-glow:hover .glow-path-2 {
          animation: glowStroke2 2s ease-in-out infinite;
          animation-delay: 0.4s;
        }

        /* Path 1 animation */
        @keyframes glowStroke1 {
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

        /* Path 2 animation */
        @keyframes glowStroke2 {
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
