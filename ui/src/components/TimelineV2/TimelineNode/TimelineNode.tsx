import { useEffect, useState } from 'react';
import * as AccordionPrimitive from '@radix-ui/react-accordion';

import TimelineItemHeader from '@/components/AccordionTimeline/TimelineItemHeader';
import Button from '@/components/Button/Button';
import CodeBlock from '@/components/Code/CodeBlock';
import MetadataItem from '@/components/Metadata/MetadataItem';
import { IconChevron } from '@/icons/Chevron';
import { type HistoryNode } from '../historyParser/index';
import renderTimelineNode from './TimelineNodeRenderer';

type Props = {
  getOutput: (historyItemID: string) => Promise<string>;
  node: HistoryNode;
};

export function TimelineNode({ getOutput, node }: Props) {
  const { icon, badge, name, metadata } = renderTimelineNode(node);
  const isExpandable = node.scope === 'step';

  return (
    <AccordionPrimitive.Item
      className="relative border-t border-slate-800/50"
      disabled={!isExpandable}
      value={node.groupID}
    >
      <AccordionPrimitive.Header className="flex gap-2 py-6">
        <div className="flex-1 z-10">
          <TimelineItemHeader icon={icon} badge={badge} title={name} metadata={metadata} />
        </div>

        {isExpandable && (
          <AccordionPrimitive.Trigger asChild>
            <Button
              className="group"
              icon={
                <IconChevron className="group-data-[state=open]:-rotate-180 transform-90 transition-transform duration-500 text-slate-500" />
              }
            />
          </AccordionPrimitive.Trigger>
        )}
      </AccordionPrimitive.Header>

      <AccordionPrimitive.Content>
        <Content getOutput={getOutput} node={node} />
      </AccordionPrimitive.Content>
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
  const output = useOutput({ getOutput, outputItemID: node.outputItemID });

  let durationMS: number | undefined;
  if (node.startedAt && node.endedAt) {
    durationMS = node.endedAt.getTime() - node.startedAt.getTime();
  }

  return (
    <>
      <div className="flex flex-grow">
        {node.startedAt && (
          <MetadataItem
            className="grow"
            label="Started at"
            value={node.scheduledAt.toLocaleString()}
          />
        )}

        {node.endedAt && (
          <MetadataItem className="grow" label="Ended at" value={node.endedAt.toLocaleString()} />
        )}

        {durationMS && (
          <MetadataItem className="grow" label="Duration" value={`${durationMS} ms`} />
        )}
      </div>

      <div>{output}</div>
    </>
  );
}

function useOutput({
  getOutput,
  outputItemID,
}: {
  getOutput: (historyItemID: string) => Promise<string>;
  outputItemID?: string;
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
        setOutput(<CodeBlock tabs={[{ label: 'Output', content: data }]} />);
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
