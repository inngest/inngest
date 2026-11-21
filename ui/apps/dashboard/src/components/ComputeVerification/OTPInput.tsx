import { useRef } from 'react';
import { Input } from '@inngest/components/Forms/Input';

export function OTPInput({
  value,
  onChange,
  disabled,
  errorID,
}: {
  value: string[];
  onChange: (value: string[]) => void;
  disabled?: boolean;
  errorID?: string;
}) {
  const inputs = useRef<Array<HTMLInputElement | null>>([]);
  function focus(index: number) {
    const input = inputs.current[Math.max(0, Math.min(5, index))];
    input?.focus();
    input?.select();
  }
  function insert(text: string, index: number) {
    const digits = text.replace(/\D/g, '').slice(0, 6);
    if (!digits) return;
    const start = digits.length === 6 ? 0 : index;
    const next = [...value];
    digits.split('').forEach((digit, offset) => {
      if (start + offset < 6) next[start + offset] = digit;
    });
    onChange(next);
    focus(start + digits.length);
  }
  return (
    <div
      role="group"
      aria-label="Six-digit verification code"
      className="grid grid-cols-6 gap-2 sm:gap-3"
    >
      {Array.from({ length: 6 }, (_, index) => (
        <Input
          key={index}
          ref={(element) => {
            inputs.current[index] = element;
          }}
          aria-label={`Digit ${index + 1}`}
          aria-invalid={Boolean(errorID)}
          aria-describedby={errorID}
          inputMode="numeric"
          autoComplete={index === 0 ? 'one-time-code' : 'off'}
          pattern="[0-9]*"
          disabled={disabled}
          value={value[index] ?? ''}
          placeholder="0"
          className="h-16 min-w-0 px-0 text-center text-xl disabled:opacity-50"
          onFocus={(event) => event.currentTarget.select()}
          onChange={(event) => {
            if (!event.target.value) {
              onChange(value.map((digit, i) => (i === index ? '' : digit)));
            } else insert(event.target.value, index);
          }}
          onPaste={(event) => {
            event.preventDefault();
            insert(event.clipboardData.getData('text'), index);
          }}
          onKeyDown={(event) => {
            if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
              event.preventDefault();
              focus(index + (event.key === 'ArrowLeft' ? -1 : 1));
            } else if (event.key === 'Home' || event.key === 'End') {
              event.preventDefault();
              focus(event.key === 'Home' ? 0 : 5);
            } else if (event.key === 'Backspace') {
              event.preventDefault();
              const target = value[index] ? index : Math.max(0, index - 1);
              onChange(value.map((digit, i) => (i === target ? '' : digit)));
              focus(target);
            } else if (event.key === 'Delete') {
              event.preventDefault();
              onChange(value.map((digit, i) => (i === index ? '' : digit)));
            }
          }}
        />
      ))}
    </div>
  );
}
