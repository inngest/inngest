import { useState } from 'react';
import { RiLightbulbLine } from '@remixicon/react';

import { AITrace } from '../AI/AITrace';
import { Button } from '../Button';
import { Pill } from '../Pill';
import { NewIO } from '../RunDetailsV3/NewIO';
import { Tabs } from '../RunDetailsV3/Tabs';
import { StatusDot } from '../Status/StatusDot';
import { getStatusTextClass } from '../Status/statusClasses';
import { cn } from '../utils/classNames';

export type StepHistoryProps = {
  id: string;
  dateStarted: Date;
  status: string;
  tagCount: number;
  input: any;
  output: any;
  aiOutput: any;
  defaultOpen?: boolean;
};

export const StepHistory = ({
  id,
  dateStarted,
  status,
  tagCount,
  input,
  output,
  aiOutput,
  defaultOpen = false,
}: StepHistoryProps) => {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div className="flex flex-col">
      <div
        className="border-muted flex cursor-pointer flex-row items-center justify-between gap-2 border-b px-4 py-3"
        onClick={() => setOpen(!open)}
      >
        <div className="text-muted w-[30%] whitespace-nowrap text-sm">
          {dateStarted.toLocaleString()}
        </div>

        <div
          className={cn(
            'flex w-[30%] flex-row items-center gap-2 text-sm',
            getStatusTextClass(status)
          )}
        >
          <StatusDot status={status} className="h-2.5 w-2.5" />
          {status}
        </div>
        <div className="w-[20%]">
          <Pill appearance="outlined" kind="primary">
            <div className="flex flex-row items-center gap-1">
              <RiLightbulbLine className="text-muted h-2.5 w-2.5" />
              {tagCount}
            </div>
          </Pill>
        </div>

        <div className="flex w-[20%] justify-end">
          <Button
            kind="secondary"
            appearance="outlined"
            size="small"
            label="View version"
            className="text-muted text-xs"
            onClick={(e) => {
              e.stopPropagation();
              console.log('view version');
            }}
          />
        </div>
      </div>
      {open && (
        <div className="flex w-full flex-col">
          <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4 px-4 py-6">
            {aiOutput && <AITrace aiOutput={aiOutput} />}
          </div>
          <Tabs
            defaultActive={'input'}
            tabs={[
              ...(input
                ? [
                    {
                      label: 'Input',
                      id: 'input',
                      node: <NewIO raw={JSON.stringify(input, null, 2)} title="Input" />,
                    },
                  ]
                : []),
              ...(output
                ? [
                    {
                      label: 'Output',
                      id: 'output',
                      node: <NewIO title="output" raw={JSON.stringify(output, null, 2)} />,
                    },
                  ]
                : []),

              {
                label: 'Tools',
                id: 'tools',
                node: <NewIO title="Tools" raw='{"tools": "coming soon..."}' />,
              },

              {
                label: 'State',
                id: 'state',
                node: <NewIO title="State" raw='{"state": "coming soon..."}' />,
              },
            ]}
          />
        </div>
      )}
    </div>
  );
};
