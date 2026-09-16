import { Pill } from '@inngest/components/Pill/Pill';
import { RiErrorWarningLine } from '@remixicon/react';

type AlertPillProps = {
  children: React.ReactNode;
  action?: React.ReactNode;
};

export function AlertPill({ children, action }: AlertPillProps) {
  return (
    <Pill
      className="text-alwaysWhite h-7 gap-2 rounded-full bg-gradient-to-r from-[rgb(var(--color-ruby-500))] to-[rgb(var(--color-ruby-800))] px-1.5 text-sm font-medium leading-5 shadow-[inset_0_0_0.1px_1px_rgba(0,0,0,0.17)]"
      icon={<RiErrorWarningLine className="h-[18px] w-[18px]" />}
      iconSide="left"
      action={action}
    >
      {children}
    </Pill>
  );
}
