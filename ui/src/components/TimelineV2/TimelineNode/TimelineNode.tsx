import { useEffect, useState } from 'react';
import * as AccordionPrimitive from '@radix-ui/react-accordion';
import { AnimatePresence, motion } from 'framer-motion';

import TimelineItemHeader from '@/components/AccordionTimeline/TimelineItemHeader';
import Button from '@/components/Button/Button';
import RunOutputCard from '@/components/Function/RunOutput';
import MetadataGrid from '@/components/Metadata/MetadataGrid';
import { IconChevron } from '@/icons/Chevron';
import classNames from '@/utils/classnames';
import { formatMilliseconds } from '@/utils/date';
import { type HistoryNode } from '../historyParser/index';
import renderTimelineNode from './TimelineNodeRenderer';

type Props = {
  getOutput: (historyItemID: string) => Promise<string>;
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
          'absolute w-px bg-slate-800 top-0 left-[0.85rem]',
          position === 'first' && 'top-[1.8rem] h-[calc(100%-1.8rem)]',
          position === 'last' && 'h-[1.8rem]',
          position === 'middle' && 'h-full',
        )}
        aria-hidden="true"
      />
      <AccordionPrimitive.Header className="flex gap-2 py-6">
        <div className="flex-1 z-10">
          <TimelineItemHeader icon={icon} badge={badge} title={name} metadata={metadata} />
        </div>

        {isExpandable && (
          <AccordionPrimitive.Trigger asChild onClick={() => toggleItem(node.groupID)}>
            <Button
              className="group"
              icon={
                <IconChevron className="group-data-[state=open]:-rotate-180 transform-90 transition-transform duration-500 text-slate-500" />
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
  getOutput: (historyItemID: string) => Promise<string>;
  node: HistoryNode;
}) {
  const output = useOutput({ getOutput, outputItemID: node.outputItemID, status: node.status });

  let durationMS: number | undefined;
  if (node.startedAt && node.endedAt) {
    durationMS = node.endedAt.getTime() - node.startedAt.getTime();
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
  getOutput: (historyItemID: string) => Promise<string>;
  outputItemID?: string;
  status: HistoryNode['status'];
}): React.ReactNode | undefined {
  const [output, setOutput] = useState<React.ReactNode>(undefined);

  useEffect(() => {
    if (!outputItemID) {
      return;
    }

    (async () => {
      setOutput('Loading...');

      try {
        const data = await getOutput(outputItemID);
        setOutput(<RunOutputCard content={data} status={status} />);
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
