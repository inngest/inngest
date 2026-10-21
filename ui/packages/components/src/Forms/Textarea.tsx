import { cn } from '../utils/classNames';

export interface TextareaProps {
  value: string;
  onChange?: (value: string) => void;
  placeholder?: string;
  readOnly?: boolean;
  className?: string;
  'aria-label'?: string;
  rows?: number;
  required?: boolean;
}

export function Textarea({
  value,
  onChange,
  placeholder,
  rows = 3,
  required,
  readOnly,
  className,
  'aria-label': ariaLabel,
}: TextareaProps) {
  return (
    <textarea
      placeholder={placeholder}
      value={value}
      onChange={(e) => {
        onChange?.(e.currentTarget.value);
      }}
      rows={rows}
      required={required}
      readOnly={readOnly}
      aria-label={ariaLabel}
      className={cn(
        'bg-canvasBase text-basis border-muted placeholder-disabled focus:outline-primary-moderate focus:border-muted w-full rounded-md border p-3 text-sm outline-2 transition-all focus:outline focus:ring-0',
        readOnly && 'bg-canvasSubtle border-subtle focus:border-subtle',
        className,
      )}
    />
  );
}
