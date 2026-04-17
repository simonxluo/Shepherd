import { useState, useEffect } from 'react';
import { cn } from '@/lib/utils';

interface NumberInputProps {
  value: number | undefined;
  onChange: (value: number) => void;
  disabled?: boolean;
  min?: number;
  max?: number;
  step?: number;
  placeholder?: string;
  className?: string;
  allowNegative?: boolean;
  allowMinusOne?: boolean;
}

export function NumberInput({
  value,
  onChange,
  disabled,
  min,
  max,
  step = 1,
  placeholder,
  className = '',
  allowNegative = false,
  allowMinusOne = false,
}: NumberInputProps) {
  const [inputValue, setInputValue] = useState(String(value ?? ''));
  const [error, setError] = useState('');

  // 同步外部 value 变化
  useEffect(() => {
    if (value !== undefined && String(value) !== inputValue) {
      setInputValue(String(value));
      setError('');
    }
  }, [value]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setInputValue(newValue);

    // 空值处理
    if (newValue === '') {
      setError('');
      return;
    }

    // 验证数字
    const num = Number(newValue);
    if (isNaN(num)) {
      setError('请输入有效数字');
      return;
    }

    // 验证范围
    if (min !== undefined && num < min && !(allowMinusOne && num === -1) && !(allowNegative && num < 0)) {
      setError(`最小值为 ${min}`);
      return;
    }
    if (max !== undefined && num > max) {
      setError(`最大值为 ${max}`);
      return;
    }

    // 特殊值验证
    if (allowMinusOne && num === -1) {
      setError('');
      onChange(-1);
      return;
    }

    if (allowNegative && num < 0) {
      setError('');
      onChange(num);
      return;
    }

    if (!allowNegative && !allowMinusOne && num < 0) {
      setError('不允许负值');
      return;
    }

    setError('');
    onChange(num);
  };

  const handleBlur = () => {
    if (inputValue === '' && value !== undefined) {
      setInputValue(String(value));
      setError('');
    }
  };

  return (
    <div>
      <input
        type="number"
        value={inputValue}
        onChange={handleChange}
        onBlur={handleBlur}
        disabled={disabled}
        min={allowMinusOne ? -1 : min}
        max={max}
        step={step}
        placeholder={placeholder}
        className={cn(
          "w-full px-2 py-1.5 text-sm",
          "border-2 rounded-md",
          error ? "border-red-500 dark:border-red-500" : "border-border",
          "bg-input",
          "text-foreground",
          "focus:outline-none focus:ring-2",
          error ? "focus:ring-red-500 focus:border-red-500" : "focus:ring-blue-500 focus:border-blue-500",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          "transition-colors",
          className
        )}
      />
      {error && (
        <p className="mt-1 text-xs text-red-600 dark:text-red-400">{error}</p>
      )}
    </div>
  );
}
