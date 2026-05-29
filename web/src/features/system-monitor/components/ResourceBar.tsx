import { cn } from '@/lib/utils';

interface ResourceBarProps {
  label: string;
  value: number;
  max: number;
  percent: number;
  formatValue?: (value: number) => string;
  formatMax?: (value: number) => string;
  className?: string;
}

function getBarColor(percent: number): string {
  if (percent >= 90) return 'bg-red-500';
  if (percent >= 70) return 'bg-yellow-500';
  return 'bg-green-500';
}

export function ResourceBar({
  label,
  value,
  max,
  percent,
  formatValue,
  formatMax,
  className,
}: ResourceBarProps) {
  const displayValue = formatValue ? formatValue(value) : value.toString();
  const displayMax = formatMax ? formatMax(max) : max.toString();
  const clampedPercent = Math.min(100, Math.max(0, percent));

  return (
    <div className={cn('space-y-1', className)}>
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium text-foreground">{label}</span>
        <span className="text-muted-foreground">
          {displayValue} / {displayMax} ({clampedPercent.toFixed(1)}%)
        </span>
      </div>
      <div className="h-2.5 w-full rounded-full bg-muted">
        <div
          className={cn('h-full rounded-full transition-all duration-500', getBarColor(clampedPercent))}
          style={{ width: `${clampedPercent}%` }}
        />
      </div>
    </div>
  );
}
