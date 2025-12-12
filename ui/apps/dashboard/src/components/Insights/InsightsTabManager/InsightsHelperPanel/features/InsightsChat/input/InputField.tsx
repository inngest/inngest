import {
  forwardRef,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  type HTMLAttributes,
  type KeyboardEventHandler,
} from 'react';
import { cn } from '@inngest/components/utils/classNames';

import SendButton from './SendButton';

export type ChatStatus = 'idle' | 'submitted' | 'streaming' | 'error';

export type PromptInputProps = HTMLAttributes<HTMLFormElement>;

export const PromptInput = ({ className, ...props }: PromptInputProps) => (
  <form
    className={cn(
      'border-muted bg-surfaceBase w-full divide-y overflow-hidden rounded-lg border pt-3',
      className,
    )}
    {...props}
  />
);

type PromptInputTextareaProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  rows?: number;
  required?: boolean;
  disabled?: boolean;
  className?: string;
  name?: string;
  onKeyDown?: KeyboardEventHandler<HTMLTextAreaElement>;
};

export const PromptInputTextarea = forwardRef<
  HTMLTextAreaElement,
  PromptInputTextareaProps
>(({ onChange, className, rows = 3, onKeyDown, disabled, ...props }, ref) => {
  const handleKeyDown: KeyboardEventHandler<HTMLTextAreaElement> = (e) => {
    if (onKeyDown) onKeyDown(e);
    if (!e.defaultPrevented && e.key === 'Enter' && !e.shiftKey) {
      if (disabled) {
        // When submission is disabled, allow Enter to insert a newline instead of submitting
        return;
      }
      e.preventDefault();
      e.currentTarget.form?.requestSubmit();
    }
  };

  return (
    <textarea
      id="user-input"
      ref={ref}
      rows={rows}
      onKeyDown={handleKeyDown}
      className={cn(
        'bg-surfaceBase text-basis placeholder-disabled focus:outline-primary-moderate w-full rounded-sm border-none p-3 text-sm outline-0 ring-0 transition-all focus:border-none focus:outline focus:ring-0 focus-visible:border-none focus-visible:outline-0 focus-visible:ring-0',
        className,
      )}
      {...props}
      onChange={(e) => onChange(e.currentTarget.value)}
    />
  );
});

PromptInputTextarea.displayName = 'PromptInputTextarea';

export const ResponsivePromptInput = ({
  value,
  onChange,
  onSubmit,
  placeholder = 'Ask insights agent to query...',
  disabled = false,
  className,
}: {
  value: string;
  onChange: (value: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const resizeTextarea = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;

    // Compute vertical chrome to include in height clamping
    const style = window.getComputedStyle(el);
    const lineHeight = parseFloat(style.lineHeight || '0');
    const paddingY =
      parseFloat(style.paddingTop || '0') +
      parseFloat(style.paddingBottom || '0');
    const borderY =
      parseFloat(style.borderTopWidth || '0') +
      parseFloat(style.borderBottomWidth || '0');

    // Define min/max rows similar to `prompt-input.tsx` usage
    const minRows = 1; // start compact; grows smoothly
    const maxRows = 10; // ~ `max-h-[30lh]`

    const minHeight = Math.max(0, lineHeight * minRows + paddingY + borderY);
    const maxHeight = Math.max(
      minHeight,
      lineHeight * maxRows + paddingY + borderY,
    );

    // Measure content height
    el.style.height = 'auto';
    const contentHeight = el.scrollHeight;

    const next = Math.min(Math.max(contentHeight, minHeight), maxHeight);
    // Set explicit pixel height for animating height
    el.style.height = `${next}px`;
  }, []);

  useLayoutEffect(() => {
    resizeTextarea();
  }, [value, resizeTextarea]);

  useEffect(() => {
    const el = textareaRef.current;
    if (!el || typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver(() => resizeTextarea());
    ro.observe(el);
    return () => ro.disconnect();
  }, [resizeTextarea]);

  return (
    <PromptInput onSubmit={onSubmit} className={className}>
      <div className="flex items-end gap-2 px-0 pb-0">
        <div className="flex-1">
          <PromptInputTextarea
            ref={textareaRef}
            value={value}
            onChange={onChange}
            placeholder={placeholder}
            disabled={disabled}
            rows={2}
            className="max-h-[30lh] w-full resize-none overflow-y-auto px-4 pb-2 pt-0 leading-6 transition-[height] duration-200 ease-out placeholder:text-base"
          />
        </div>
        <div className="flex items-center justify-end px-3 pb-3">
          <div className="flex items-center gap-2">
            <SendButton
              onClick={onSubmit}
              disabled={disabled || !value.trim()}
            />
          </div>
        </div>
      </div>
    </PromptInput>
  );
};
