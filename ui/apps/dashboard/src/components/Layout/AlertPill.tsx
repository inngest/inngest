import { Pill } from '@inngest/components/Pill/Pill';
import { cn } from '@inngest/components/utils/classNames';
import { RiErrorWarningLine } from '@remixicon/react';

export type AlertPillKind = 'error' | 'warning' | 'caution';

type AlertPillProps = {
  children: React.ReactNode;
  action?: React.ReactNode;
  kind?: AlertPillKind;
};

const kindStyles: Record<AlertPillKind, { gradient: string; text: string }> = {
  error: {
    gradient:
      'from-[rgb(var(--color-ruby-500))] to-[rgb(var(--color-ruby-800))]',
    text: 'text-alwaysWhite',
  },
  warning: {
    gradient:
      'from-[rgb(var(--color-honey-500))] to-[rgb(var(--color-honey-800))]',
    text: 'text-alwaysWhite',
  },
  caution: {
    gradient:
      'from-[rgb(var(--color-honey-300))] to-[rgb(var(--color-honey-400))]',
    text: 'text-alwaysBlack',
  },
};

export function AlertPill({
  children,
  action,
  kind = 'error',
}: AlertPillProps) {
  return (
    <Pill
      className={cn(
        'h-7 gap-2 rounded-full bg-gradient-to-r px-1.5 text-sm font-medium leading-5 shadow-[inset_0_0_0.1px_1px_rgba(0,0,0,0.17)]',
        kindStyles[kind].gradient,
        kindStyles[kind].text,
      )}
      icon={<RiErrorWarningLine className="h-[18px] w-[18px]" />}
      iconSide="left"
      action={action}
    >
      {children}
    </Pill>
  );
}
