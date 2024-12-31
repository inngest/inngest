'use client';

import { useEffect, useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { MetadataGrid } from '@inngest/components/Metadata';
import { OutputCard } from '@inngest/components/OutputCard';
import { renderStepMetadata } from '@inngest/components/RunDetails/stepMetadataRenderer';
import { type HistoryNode } from '@inngest/components/utils/historyParser';
import { isEndStatus, type Status } from '@inngest/components/utils/historyParser/types';
import * as AccordionPrimitive from '@radix-ui/react-accordion';
import { RiArrowDownSLine } from '@remixicon/react';
import { AnimatePresence, motion } from 'framer-motion';

import type { Timeline } from '..';
import { TimelineNodeHeader } from './TimelineNodeHeader';
import { renderTimelineNode } from './TimelineNodeRenderer';

type Props = {
  getOutput: (historyItemID: string) => Promise<string | undefined>;
  node: HistoryNode;
  children?: React.ReactNode;
  isAttempt?: boolean;
  navigateToRun: React.ComponentProps<typeof Timeline>['navigateToRun'];
};

export function TimelineNode({ getOutput, node, children, isAttempt, navigateToRun }: Props) {
  const { icon, badge, name, metadata, runLink } = renderTimelineNode({ node, isAttempt });
  const isExpandable = node.scope === 'step';
  const [openItems, setOpenItems] = useState<string[]>([]);

  const runLinkNode = useMemo(() => {
    if (!runLink) {
      return null;
    }

    return navigateToRun(runLink);
  }, [...Object.values(runLink ?? {})]);

  const toggleItem = (itemValue: string) => {
    if (openItems.includes(itemValue)) {
      setOpenItems(openItems.filter((value) => value !== itemValue));
    } else {
      setOpenItems([...openItems, itemValue]);
    }
  };
  const value = `${node.groupID}${isAttempt ? `/attempt${node.attempt}` : ''}`;

  return (
    <AccordionPrimitive.Item
      className="border-subtle border-t"
      disabled={!isExpandable}
      value={value}
    >
      <AccordionPrimitive.Header className="bg-canvasBase flex items-start gap-2 py-6">
        <div className="z-10 flex-1">
          <TimelineNodeHeader icon={icon} badge={badge} title={name} metadata={metadata} />
        </div>

        {isExpandable && (
          <AccordionPrimitive.Trigger
            asChild
            onClick={() => toggleItem(value)}
            className="bg-canvasBase hover:bg-canvasSubtle group"
          >
            <Button
              className="group"
              appearance="outlined"
              kind="secondary"
              icon={
                <RiArrowDownSLine className="transform-90 bg-canvasBase group-hover:bg-canvasSubtle text-subtle transition-transform duration-500 group-data-[state=open]:-rotate-180" />
              }
            />
          </AccordionPrimitive.Trigger>
        )}
      </AccordionPrimitive.Header>
      <AnimatePresence>
        {openItems.includes(value) && (
          <AccordionPrimitive.Content className="ml-8" forceMount>
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
              <Content
                getOutput={getOutput}
                node={node}
                isAttempt={isAttempt}
                links={[runLinkNode]}
              />
              {children}
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
  isAttempt,
  links,
}: {
  getOutput: (historyItemID: string) => Promise<string | undefined>;
  node: HistoryNode;
  isAttempt?: boolean;
  links?: React.ReactNode[];
}) {
  const output = useOutput({
    getOutput,
    outputItemID: node.outputItemID,
    nodeStatus: node.status,
  });

  const metadataItems = renderStepMetadata({ node, isAttempt });

  return (
    <>
      {links?.length ? <div className="flex flex-row justify-end gap-x-5 pb-5">{links}</div> : null}

      <div className="pb-5">
        <MetadataGrid metadataItems={metadataItems} />
      </div>

      {output && <div className="pb-5">{output}</div>}
    </>
  );
}

function useOutput({
  getOutput,
  outputItemID,
  nodeStatus,
}: {
  getOutput: (historyItemID: string) => Promise<string | undefined>;
  outputItemID?: string;
  nodeStatus: Status;
}): React.ReactNode | undefined {
  const [output, setOutput] = useState<React.ReactNode>(undefined);

  useEffect(() => {
    if (!outputItemID) {
      return;
    }

    // We should only fetch output if the node has ended or errored ("errored"
    // means there will be a retry). There are some edge cases where a node can
    // have an output item ID but not be in an end state
    if (!isEndStatus(nodeStatus) && nodeStatus != 'errored') {
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

        setOutput(<OutputCard content={data} isSuccess={nodeStatus === 'completed'} />);
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
