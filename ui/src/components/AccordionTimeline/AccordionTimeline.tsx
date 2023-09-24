import * as AccordionPrimitive from '@radix-ui/react-accordion';

import Button from '@/components/Button/Button';
import { IconChevron } from '@/icons';
import classNames from '@/utils/classnames';

type AccordionTimelineProps = {
  timelineItems: AccordionTimelineItemProps[];
};

export default function AccordionTimeline({ timelineItems }: AccordionTimelineProps) {
  return (
    <AccordionPrimitive.Root
      type="multiple"
      className="text-slate-100 w-full last:border-b last:border-slate-800/50"
    >
      {timelineItems &&
        timelineItems.map((item) => {
          if (!item) return <p></p>;
          const { id, header, expandable, position, content } = item;
          return (
            <AccordionTimelineItem
              id={id}
              header={header}
              expandable={expandable}
              position={position}
              content={content}
            />
          );
        })}
    </AccordionPrimitive.Root>
  );
}

type AccordionTimelineItemProps = {
  header: React.ReactNode;
  expandable?: boolean;
  position?: 'first' | 'last' | 'middle';
  content?: React.ReactNode;
  id: string;
};

export function AccordionTimelineItem({
  id,
  header,
  expandable = true,
  position = 'middle',
  content,
}: AccordionTimelineItemProps) {
  return (
    <AccordionPrimitive.Item
      key={id}
      value={id}
      disabled={!expandable}
      className="relative border-t border-slate-800/50"
    >
      <span
        className={classNames(
          'absolute w-px bg-slate-800 top-0 left-[0.85rem]',
          position === 'first' && 'top-[1.8rem] h-[calc(100%-1.8rem)]',
          position === 'last' && 'h-[1.8rem]',
          position === 'middle' && 'h-full',
        )}
        aria-hidden="true"
      />
      <AccordionPrimitive.Header className="flex gap-2 py-6">
        <div className="flex-1 z-10">{header}</div>
        {expandable && (
          <>
            <div className="border-r border-slate-800/50" />
            <AccordionPrimitive.Trigger asChild>
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
      <AccordionPrimitive.Content className="data-[state=open]:animate-slide-down data-[state=closed]:animate-slide-up overflow-hidden pl-9">
        {content}
      </AccordionPrimitive.Content>
    </AccordionPrimitive.Item>
  );
}
