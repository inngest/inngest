'use client';

import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { MetadataGrid } from '@inngest/components/Metadata';
import { OutputCard } from '@inngest/components/OutputCard';
import { IconChevron } from '@inngest/components/icons/Chevron';
import { classNames } from '@inngest/components/utils/classNames';
import { formatMilliseconds } from '@inngest/components/utils/date';
import { type HistoryNode } from '@inngest/components/utils/historyParser';
import * as AccordionPrimitive from '@radix-ui/react-accordion';
import { AnimatePresence, motion } from 'framer-motion';

import { TimelineNodeHeader } from './TimelineNodeHeader';
import { renderTimelineNode } from './TimelineNodeRenderer';

type Props = {
  getOutput: (historyItemID: string) => Promise<string | undefined>;
  node: HistoryNode;
  position: 'first' | 'last' | 'middle';
};

export function TimelineNode({ position, getOutput, node }: Props) {
  const { icon, badge, name, metadata } = renderTimelineNode(node);
  const isExpandable = node.scope === 'step';
  const [openItems, setOpenItems] = useState<string[]>([]);

  const toggleItem = (itemValue: string) => {
    if (openItems.includes(itemValue)) {
      setOpenItems(openItems.filter((value) => value !== itemValue));
    } else {
      setOpenItems([...openItems, itemValue]);
    }
  };

  return (
    <AccordionPrimitive.Item
      className="relative border-t border-slate-800/50"
      disabled={!isExpandable}
      value={node.groupID}
    >
      <span
        className={classNames(
          'absolute left-[0.85rem] top-0 w-px bg-slate-800',
          position === 'first' && 'top-[1.8rem] h-[calc(100%-1.8rem)]',
          position === 'last' && 'h-[1.8rem]',
          position === 'middle' && 'h-full'
        )}
        aria-hidden="true"
      />
      <AccordionPrimitive.Header className="flex items-start gap-2 py-6">
        <div className="z-10 flex-1">
          <TimelineNodeHeader icon={icon} badge={badge} title={name} metadata={metadata} />
        </div>

        {isExpandable && (
          <AccordionPrimitive.Trigger asChild onClick={() => toggleItem(node.groupID)}>
            <Button
              className="group"
              icon={
                <IconChevron className="transform-90 text-slate-500 transition-transform duration-500 group-data-[state=open]:-rotate-180" />
              }
            />
          </AccordionPrimitive.Trigger>
        )}
      </AccordionPrimitive.Header>
      <AnimatePresence>
        {openItems.includes(node.groupID) && (
          <AccordionPrimitive.Content className="ml-9" forceMount>
            <motion.div
              initial={{ y: -20, opacity: 0.2 }}
              animate={{ y: 0, opacity: 1 }}
              exit={{
                y: -20,
                opacity: 0.2,
                transition: { duration: 0.2, type: 'tween' },
              }}
              transition={{
                duration: 0.15,
                type: 'tween',
              }}
            >
              <Content getOutput={getOutput} node={node} />
            </motion.div>
          </AccordionPrimitive.Content>
        )}
      </AnimatePresence>
    </AccordionPrimitive.Item>
  );
}

function Content({
  getOutput,
  node,
}: {
  getOutput: (historyItemID: string) => Promise<string | undefined>;
  node: HistoryNode;
}) {
  const output = useOutput({ getOutput, outputItemID: node.outputItemID, status: node.status });

  let durationMS: number | undefined;
  if (node.scheduledAt && node.endedAt) {
    durationMS = node.endedAt.getTime() - node.scheduledAt.getTime();
  }

  return (
    <>
      <div className="pb-5">
        <MetadataGrid
          metadataItems={[
            {
              label: 'Started At',
              value: node.scheduledAt ? node.scheduledAt.toLocaleString() : '-',
            },
            {
              label: 'Ended At',
              value: node.endedAt ? node.endedAt.toLocaleString() : '-',
            },
            {
              label: 'Duration',
              value: durationMS ? formatMilliseconds(durationMS) : '-',
            },
          ]}
        />
      </div>

      {output && <div className="pb-5">{output}</div>}
    </>
  );
}

function useOutput({
  getOutput,
  outputItemID,
  status,
}: {
  getOutput: (historyItemID: string) => Promise<string | undefined>;
  outputItemID?: string;
  status: HistoryNode['status'];
}): React.ReactNode | undefined {
  const [output, setOutput] = useState<React.ReactNode>(undefined);

  useEffect(() => {
    if (!outputItemID) {
      return;
    }
    if (status !== 'completed' && status !== 'failed') {
      return;
    }

    (async () => {
      setOutput('Loading...');

      try {
        const data = await getOutput(outputItemID);
        if (data === undefined) {
          setOutput(undefined);
          return;
        }

        setOutput(<OutputCard content={data} type={status} />);
      } catch (e) {
        let text = 'Error loading';
        if (e instanceof Error) {
          text += `: ${e.message}`;
        }
        setOutput(text);
      }
    })();
  }, [getOutput, outputItemID]);

  return output;
}
