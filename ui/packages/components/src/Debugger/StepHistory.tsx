import { useState } from 'react';
import { RiLightbulbLine } from '@remixicon/react';

import { AITrace } from '../AI/AITrace';
import { Button } from '../Button';
import { Pill } from '../Pill';
import { IO } from '../RunDetailsV3/IO';
import { Tabs } from '../RunDetailsV3/Tabs';
import type { RunTraceSpan } from '../SharedContext/useGetDebugRun';
import { StatusDot } from '../Status/StatusDot';
import { getStatusTextClass } from '../Status/statusClasses';
import { cn } from '../utils/classNames';
import { exampleAiOutput, exampleInput, exampleOutput } from './History';

export type StepHistoryProps = {
  debugRun: RunTraceSpan;
  defaultOpen?: boolean;
};

export const StepHistory = ({ debugRun, defaultOpen = false }: StepHistoryProps) => {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div className="flex flex-col">
      <div
        className="border-muted flex cursor-pointer flex-row items-center justify-between gap-2 border-b px-4 py-3"
        onClick={() => setOpen(!open)}
      >
        <div className="text-muted w-[30%] whitespace-nowrap text-sm">
          {debugRun.startedAt?.toLocaleString()}
        </div>

        <div
          className={cn(
            'flex w-[30%] flex-row items-center gap-2 text-sm',
            getStatusTextClass(debugRun.status)
          )}
        >
          <StatusDot status={debugRun.status} className="h-2.5 w-2.5" />
          {debugRun.status}
        </div>
        <div className="w-[20%]">
          <Pill appearance="outlined" kind="primary">
            <div className="flex flex-row items-center gap-1">
              <RiLightbulbLine className="text-muted h-2.5 w-2.5" />

              {Math.floor(Math.random() * 100)}
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
            {exampleAiOutput && <AITrace aiOutput={exampleAiOutput} />}
          </div>
          <Tabs
            defaultActive={'input'}
            tabs={[
              ...(exampleInput
                ? [
                    {
                      label: 'Input',
                      id: 'input',
                      node: (
                        <IO
                          raw={JSON.stringify(exampleInput, null, 2)}
                          title="Input"
                          parsed={true}
                        />
                      ),
                    },
                  ]
                : []),
              ...(exampleOutput
                ? [
                    {
                      label: 'Output',
                      id: 'output',
                      node: (
                        <IO
                          title="output"
                          raw={JSON.stringify(exampleOutput, null, 2)}
                          parsed={true}
                        />
                      ),
                    },
                  ]
                : []),

              {
                label: 'Tools',
                id: 'tools',
                node: <IO title="Tools" raw='{"tools": "coming soon..."}' parsed={true} />,
              },

              {
                label: 'State',
                id: 'state',
                node: <IO title="State" raw='{"state": "coming soon..."}' parsed={true} />,
              },
            ]}
          />
        </div>
      )}
    </div>
  );
};
