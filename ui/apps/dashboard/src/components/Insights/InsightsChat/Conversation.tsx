'use client';

import { type ComponentProps } from 'react';
import { Button } from '@inngest/components/Button';
import { cn } from '@inngest/components/utils/classNames';
import { RiArrowDownLine } from '@remixicon/react';

import { useStickToBottom } from '@/components/Insights/InsightsChat/hooks/use-stick-to-bottom';

export const Conversation = ({ className, ...props }: ComponentProps<'div'>) => {
  const { scrollRef, isAtBottom, scrollToBottom } = useStickToBottom();

  return (
    <div className={cn('relative min-h-0 flex-1', className)} {...props}>
      <div ref={scrollRef} className="h-full overflow-y-auto">
        {props.children}
      </div>
      {!isAtBottom && (
        <ConversationScrollButton
          onClick={scrollToBottom}
          className="absolute bottom-4 left-1/2 -translate-x-1/2"
        />
      )}
    </div>
  );
};

export const ConversationContent = ({ className, ...props }: ComponentProps<'div'>) => (
  <div className={cn('mx-4 pb-1 pt-4', className)} {...props} />
);

export const ConversationScrollButton = ({
  className,
  ...props
}: React.ComponentProps<typeof Button>) => {
  return (
    <Button
      className={cn('rounded-full', className)}
      appearance="outlined"
      icon={<RiArrowDownLine className="text-subtle size-4" />}
      {...props}
    />
  );
};
