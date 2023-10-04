import { useState, type PropsWithChildren } from 'react';
import * as AccordionPrimitive from '@radix-ui/react-accordion';

import Button from '@/components/Button/Button';
import { IconChevron } from '@/icons';
import CodeBlock from '../Code/CodeBlock';

export default function AccordionTimeline({ children }: PropsWithChildren) {
  return (
    <AccordionPrimitive.Root
      type="multiple"
      className="text-slate-100 w-full last:border-b last:border-slate-800/50"
    >
      {children}
    </AccordionPrimitive.Root>
  );
}

type AccordionTimelineItemProps = {
  getContent?: () => Promise<string>;
  header: React.ReactNode;
  id: string;
};

export function AccordionTimelineItem({ getContent, id, header }: AccordionTimelineItemProps) {
  const [content, setContent] = useState<React.ReactNode>(undefined);

  let onExpand: (() => void) | undefined;
  if (getContent) {
    onExpand = async () => {
      if (content) {
        // Already loaded.
        return
      }

      setContent('Loading...');

      try {
        const data = await getContent();
        setContent(<CodeBlock tabs={[{ label: 'Output', content: data }]} />);
      } catch (e) {
        let text = 'Error loading';
        if (e instanceof Error) {
          text += `: ${e.message}`;
        }
        setContent(text);
      }
    };
  }

  return (
    <AccordionPrimitive.Item
      value={id}
      disabled={!getContent}
      className="relative border-t border-slate-800/50"
    >
      <AccordionPrimitive.Header className="flex gap-2 py-6">
        <div className="flex-1 z-10">{header}</div>
        {onExpand && (
          <>
            <div className="border-r border-slate-800/50" />
            <AccordionPrimitive.Trigger asChild onClick={onExpand}>
              <Button
                className="group"
                icon={
                  <IconChevron className="group-data-[state=open]:-rotate-180 transform-90 transition-transform duration-500 text-slate-500" />
                }
              />
            </AccordionPrimitive.Trigger>
          </>
        )}
      </AccordionPrimitive.Header>

      <AccordionPrimitive.Content>{content}</AccordionPrimitive.Content>
    </AccordionPrimitive.Item>
  );
}
