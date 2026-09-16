import { useState, type ClipboardEvent } from 'react';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { RiEyeLine, RiEyeOffLine } from '@remixicon/react';

type Props = {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  error?: string;
};

export function SecretValueInput({
  id,
  label,
  value,
  onChange,
  disabled,
  error,
}: Props) {
  const [visible, setVisible] = useState(false);
  const multiline = /[\r\n]/.test(value);

  function paste(event: ClipboardEvent<HTMLInputElement>) {
    const pasted = event.clipboardData.getData('text/plain');
    if (/[\r\n]/.test(pasted)) {
      // Native inputs strip newlines. Preserve pasted certificates/keys intact.
      event.preventDefault();
      const input = event.currentTarget;
      onChange(
        multiline
          ? pasted
          : value.slice(0, input.selectionStart ?? 0) +
              pasted +
              value.slice(input.selectionEnd ?? value.length),
      );
    }
  }

  return (
    <div className="min-w-0">
      <div className="flex items-start gap-1">
        <div className="min-w-0 flex-1">
          {visible ? (
            <textarea
              id={id}
              aria-label={label}
              aria-invalid={Boolean(error)}
              aria-describedby={error ? `${id}-error` : undefined}
              value={value}
              onChange={(event) => onChange(event.target.value)}
              disabled={disabled}
              rows={multiline ? 4 : 1}
              placeholder="Value"
              autoComplete="off"
              spellCheck={false}
              data-1p-ignore
              data-sentry-mask
              className="border-muted placeholder-disabled text-basis focus:border-active min-h-8 w-full resize-y rounded border bg-transparent px-2 py-1.5 font-mono text-xs outline-none disabled:opacity-50"
            />
          ) : (
            <Input
              id={id}
              aria-label={label}
              aria-invalid={Boolean(error)}
              aria-describedby={error ? `${id}-error` : undefined}
              type="password"
              value={multiline ? '••••••••' : value}
              onChange={(event) => onChange(event.target.value)}
              onPaste={paste}
              readOnly={multiline}
              disabled={disabled}
              placeholder="Value"
              className="font-mono text-xs"
              data-sentry-mask
            />
          )}
          {multiline && !visible && (
            <p className="text-muted mt-1 text-xs">
              Multiline value. Show to edit.
            </p>
          )}
        </div>
        <Button
          kind="secondary"
          appearance="ghost"
          size="small"
          icon={visible ? <RiEyeOffLine /> : <RiEyeLine />}
          aria-label={`${visible ? 'Hide' : 'Show'} ${label.toLowerCase()}`}
          disabled={disabled}
          onClick={() => setVisible(!visible)}
        />
      </div>
      {error && (
        <p id={`${id}-error`} className="text-error mt-1 text-xs">
          {error}
        </p>
      )}
    </div>
  );
}
