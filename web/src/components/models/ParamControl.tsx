import { useState, useRef, useEffect } from 'react';
import { ToggleLeft, ToggleRight, Info } from 'lucide-react';
import { cn } from '@/lib/utils';
import { PARAM_HELP } from './LoadModelDialogConstants';

interface ParamControlProps {
  paramKey: string;
  showToggle?: boolean;
  isLoading: boolean;
  isEnabled: boolean;
  onToggleEnabled: (paramKey: string) => void;
  activeTooltip: string | null;
  onSetActiveTooltip: React.Dispatch<React.SetStateAction<string | null>>;
}

export function ParamControl({
  paramKey,
  showToggle = true,
  isLoading,
  isEnabled,
  onToggleEnabled,
  activeTooltip,
  onSetActiveTooltip,
}: ParamControlProps) {
  const helpText = PARAM_HELP[paramKey as keyof typeof PARAM_HELP];
  const buttonRef = useRef<HTMLButtonElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });

  const updatePosition = () => {
    if (buttonRef.current && activeTooltip === paramKey) {
      const rect = buttonRef.current.getBoundingClientRect();
      setPosition({
        top: rect.top - 8,
        left: rect.left + rect.width / 2,
      });
    }
  };

  useEffect(() => {
    if (activeTooltip === paramKey) {
      updatePosition();
      const handleScroll = () => updatePosition();
      window.addEventListener('scroll', handleScroll, true);
      window.addEventListener('resize', updatePosition);
      return () => {
        window.removeEventListener('scroll', handleScroll, true);
        window.removeEventListener('resize', updatePosition);
      };
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTooltip, paramKey]);

  useEffect(() => {
    if (activeTooltip !== paramKey) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (
        buttonRef.current &&
        !buttonRef.current.contains(e.target as Node) &&
        tooltipRef.current &&
        !tooltipRef.current.contains(e.target as Node)
      ) {
        onSetActiveTooltip(() => null);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [activeTooltip, paramKey, onSetActiveTooltip]);

  const handleToggleTooltip = () => {
    onSetActiveTooltip(prev => prev === paramKey ? null : paramKey);
  };

  const handleToggleEnabled = () => {
    console.log('[ToggleEnabled] 点击按钮:', paramKey);
    onToggleEnabled(paramKey);
  };

  return (
    <div className="relative inline-flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
      {/* 启用/禁用开关 */}
      {showToggle && (
        <button
          type="button"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            console.log('[Toggle Button] onClick triggered, paramKey:', paramKey);
            console.log('[Toggle Button] isLoading:', isLoading, 'isEnabled:', isEnabled);
            console.log('[Toggle Button] event:', e);
            handleToggleEnabled();
            // 点击后移除焦点，防止焦点转移到其他元素
            (e.currentTarget as HTMLButtonElement).blur();
          }}
          onMouseDown={(e) => {
            // 防止焦点转移
            e.preventDefault();
          }}
          disabled={isLoading}
          className={cn(
            "inline-flex items-center justify-center w-6 h-6 rounded transition-all duration-200",
            "focus:outline-none focus:ring-1 focus:ring-blue-500 dark:focus:ring-blue-400",
            isEnabled
              ? "text-green-600 dark:text-green-400 hover:text-green-700 dark:hover:text-green-300 hover:bg-green-50 dark:hover:bg-green-900/20"
              : "text-gray-400 dark:text-gray-600 hover:text-gray-500 dark:hover:text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800/20",
            "disabled:opacity-50 disabled:cursor-not-allowed",
            "active:scale-95",
            "select-none"
          )}
          aria-label={isEnabled ? `禁用 ${paramKey}` : `启用 ${paramKey}`}
          title={isEnabled ? '已启用 - 点击禁用' : '已禁用 - 点击启用'}
          data-param-key={paramKey}
          data-is-enabled={String(isEnabled)}
          data-is-loading={String(isLoading)}
        >
          {isEnabled ? (
            <ToggleRight className="w-4 h-4" />
          ) : (
            <ToggleLeft className="w-4 h-4" />
          )}
        </button>
      )}

      {/* 帮助按钮 */}
      <div className="relative inline-block">
        <button
          ref={buttonRef}
          type="button"
          onClick={handleToggleTooltip}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              handleToggleTooltip();
            }
          }}
          className={cn(
            "w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium",
            "flex items-center justify-center",
            "bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600",
            "hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40",
            "hover:text-blue-600 dark:hover:text-blue-400",
            "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1",
            "transition-all duration-200 cursor-help shadow-sm hover:shadow",
            activeTooltip === paramKey && "from-blue-50 to-blue-100 dark:from-blue-900/40 dark:to-blue-800/40 text-blue-600 dark:text-blue-400",
            !isEnabled && "opacity-50"
          )}
          aria-label={`查看 ${paramKey} 的帮助说明`}
          aria-expanded={activeTooltip === paramKey}
          aria-controls={`tooltip-${paramKey}`}
        >
          ?
        </button>

        {/* Tooltip */}
        {activeTooltip === paramKey && (
          <div
            ref={tooltipRef}
            id={`tooltip-${paramKey}`}
            role="tooltip"
            className="fixed z-[100]"
            style={{
              top: `${position.top}px`,
              left: `${position.left}px`,
              transform: 'translateX(-50%) translateY(-100%)',
              animation: 'tooltipFadeIn 0.2s ease-out forwards',
            }}
          >
            <style>{`
              @keyframes tooltipFadeIn {
                from {
                  opacity: 0;
                  transform: 'translateX(-50%) translateY(-10%)';
                }
                to {
                  opacity: 1;
                  transform: 'translateX(-50%) translateY(-100%)';
                }
              }
            `}</style>
            <div className="relative mb-1.5">
              <div className="max-w-xs px-4 py-3 bg-background/95 backdrop-blur-xl rounded-xl shadow-2xl border border-white/10">
                <div className="flex items-start gap-3">
                  <Info className="w-4 h-4 text-blue-400 mt-0.5 flex-shrink-0" />
                  <p className="text-sm text-foreground leading-relaxed">
                    {helpText || '暂无说明'}
                  </p>
                </div>
              </div>

              {/* 下方箭头 */}
              <div className="absolute -bottom-1.5 left-1/2 -translate-x-1/2">
                <div className="w-0 h-0 border-l-[6px] border-l-transparent border-r-[6px] border-r-transparent border-t-[6px] border-t-gray-900/95 backdrop-blur-xl" />
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
