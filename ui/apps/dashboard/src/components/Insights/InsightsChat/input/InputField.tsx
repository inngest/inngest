'use client';

import {
  forwardRef,
  useCallback,
  useEffect,
  useRef,
  useState,
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
      'border-muted bg-surfaceBase w-full divide-y overflow-hidden rounded-lg border pt-4 shadow-sm',
      className
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

export const PromptInputTextarea = forwardRef<HTMLTextAreaElement, PromptInputTextareaProps>(
  ({ onChange, className, rows = 3, onKeyDown, ...props }, ref) => {
    const handleKeyDown: KeyboardEventHandler<HTMLTextAreaElement> = (e) => {
      if (onKeyDown) onKeyDown(e);
      if (!e.defaultPrevented && e.key === 'Enter' && !e.shiftKey) {
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
          'bg-canvasBase placeholder-disabled focus:outline-primary-moderate w-full rounded-sm border-none border-none p-3 text-sm outline-0 ring-0 transition-all focus:border-none focus:border-none focus:outline focus:ring-0 focus-visible:border-none focus-visible:outline-0 focus-visible:ring-0',
          className
        )}
        {...props}
        onChange={(e) => onChange(e.currentTarget.value)}
      />
    );
  }
);

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
  const [isExpanded, setIsExpanded] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const checkExpansion = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;
    if (value.includes('\n')) {
      if (!isExpanded) setIsExpanded(true);
      return;
    }
    const tempSpan = document.createElement('span');
    tempSpan.style.position = 'absolute';
    tempSpan.style.visibility = 'hidden';
    tempSpan.style.whiteSpace = 'nowrap';
    if (typeof window !== 'undefined') {
      const style = window.getComputedStyle(textarea);
      tempSpan.style.fontSize = style.fontSize;
      tempSpan.style.fontFamily = style.fontFamily;
      tempSpan.style.fontWeight = style.fontWeight;
      tempSpan.style.letterSpacing = style.letterSpacing;
    }
    tempSpan.textContent = value || textarea.placeholder;
    document.body.appendChild(tempSpan);
    const textWidth = tempSpan.getBoundingClientRect().width;
    document.body.removeChild(tempSpan);
    const availableWidth = textarea.getBoundingClientRect().width - 48;
    const shouldExpand = textWidth >= availableWidth;
    if (shouldExpand && !isExpanded) {
      setIsExpanded(true);
    } else if (!value.trim() && isExpanded) {
      setIsExpanded(false);
    }
  }, [value, isExpanded]);

  useEffect(() => {
    const timeoutId = setTimeout(() => {
      checkExpansion();
    }, 0);
    return () => clearTimeout(timeoutId);
  }, [checkExpansion]);

  useEffect(() => {
    if (isExpanded && textareaRef.current) {
      const textarea = textareaRef.current;
      setTimeout(() => {
        textarea.focus();
        const length = textarea.value.length;
        textarea.setSelectionRange(length, length);
      }, 0);
    }
  }, [isExpanded]);

  if (isExpanded) {
    return (
      <PromptInput onSubmit={onSubmit} className={className}>
        <div className="flex flex-col">
          <div className="relative mb-2 w-full">
            <PromptInputTextarea
              ref={textareaRef}
              rows={5}
              value={value}
              onChange={onChange}
              placeholder={placeholder}
              disabled={disabled}
              className="max-h-[30lh] min-h-[3lh] w-full resize-none px-4 py-0 pt-0 leading-6 placeholder:text-base"
            />
            <div className="from-surfaceBase pointer-events-none absolute left-0 right-0 top-0 h-3 bg-gradient-to-b to-transparent" />
            {/* Bottom gradient overlay */}
            <div className="from-surfaceBase pointer-events-none absolute bottom-0 left-0 right-0 h-3 bg-gradient-to-t to-transparent" />
          </div>
          <div className="flex items-center justify-end px-3 pb-3">
            <div className="flex items-center gap-2">
              <SendButton onClick={onSubmit} disabled={disabled || !value.trim()} />
            </div>
          </div>
        </div>
      </PromptInput>
    );
  }

  return (
    <PromptInput onSubmit={onSubmit} className={className}>
      <div className="flex h-14 items-center gap-2 px-0 pb-0">
        {/* Plus button removed */}
        <div className="flex-1">
          <PromptInputTextarea
            ref={textareaRef}
            value={value}
            onChange={onChange}
            placeholder={placeholder}
            disabled={disabled}
            className="w-full resize-none px-4 py-0 pt-3 leading-6"
          />
        </div>
        <div className="relative top-1 flex items-center justify-end px-3 pb-0">
          <div className="flex items-center gap-2 pr-0.5">
            <SendButton onClick={onSubmit} disabled={disabled || !value.trim()} />
          </div>
        </div>
      </div>
    </PromptInput>
  );
};
