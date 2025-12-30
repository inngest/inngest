import { useCallback, useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiArrowRightSLine } from '@remixicon/react';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Alert } from '../Alert';
import {
  CodeElement,
  ElementWrapper,
  IDElement,
  SkeletonElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
import { ErrorCard } from '../Error/ErrorCard';
import { InvokeModal } from '../InvokeButton';
import type { TraceResult } from '../SharedContext/useGetTraceResult';
import { useInvokeRun } from '../SharedContext/useInvokeRun';
import { usePrettyErrorBody, usePrettyJson } from '../hooks/usePrettyJson';
import { IconCloudArrowDown } from '../icons/CloudArrowDown';
import { devServerURL, useDevServer } from '../utils/useDevServer';
import { ErrorInfo } from './ErrorInfo';
import { IO } from './IO';
import { MetadataAttrs } from './MetadataAttrs';
import { Tabs } from './Tabs';
import type { Trace } from './types';

type TopInfoProps = {
  slug?: string;
  getTrigger: (runID: string) => Promise<Trigger>;
  result?: TraceResult;
  runID: string;
  resultLoading?: boolean;
  trace?: Trace;
};

export type Trigger = {
  payloads: string[];
  timestamp: string;
  eventName: string | null;
  IDs: string[];
  batchID: string | null;
  isBatch: boolean;
  cron: string | null;
};

interface ActionConfig {
  title: string;
  disabled?: boolean;
  onClick?: () => void;
}

export const actionConfigs = (
  trigger: Trigger | undefined,
  isRunning: boolean,
  send: (payload: string) => void
): ActionConfig => {
  if (!trigger) {
    return { title: 'Loading trigger' };
  }

  if (trigger.isBatch) {
    return { title: "Can't send a batch" };
  }

  if (trigger.cron) {
    return { title: "Can't send a cron" };
  }

  const payload = trigger.payloads[0];
  if (!payload) {
    console.error('Trigger has no payloads');
    return { title: 'Trigger has no payloads' };
  }

  return {
    title: isRunning
      ? 'Send event payload to running Dev Server'
      : `Dev Server is not running at ${devServerURL}`,
    disabled: !isRunning,
    onClick: () => send(payload),
  };
};

export const TopInfo = ({
  slug,
  getTrigger,
  runID,
  result,
  resultLoading,
  trace,
}: TopInfoProps) => {
  const [expanded, setExpanded] = useState(true);
  const { isRunning, send } = useDevServer();
  const { invoke, loading: invokeLoading, error: invokeError } = useInvokeRun();
  const [invokeOpen, setInvokeOpen] = useState(false);

  const {
    data: trigger,
    error,
    isPending,
    refetch,
  } = useQuery({
    queryKey: ['run-trigger', runID],
    queryFn: useCallback(() => {
      return getTrigger(runID);
    }, [getTrigger, runID]),
    retry: 3,
  });

  const prettyPayload = useMemo(() => {
    try {
      const data = trigger?.payloads?.map((p) => JSON.parse(p));
      return JSON.stringify(data, null, 2);
    } catch (e) {
      console.warn('Unable to parse payloads as JSON:', trigger?.payloads);
      return undefined;
    }
  }, [trigger?.payloads]);

  const prettyOutput = usePrettyJson(result?.data ?? '') || (result?.data ?? '');
  const prettyErrorBody = usePrettyErrorBody(result?.error);

  const type = trigger?.isBatch ? 'BATCH' : trigger?.cron ? 'CRON' : 'EVENT';

  const codeBlockActions = useMemo(() => {
    return [
      {
        label: 'Send to Dev Server',
        icon: <IconCloudArrowDown />,
        disabled: true,
        onClick: () => {},
        ...actionConfigs(trigger, isRunning, send),
      },
    ];
  }, [trigger, isRunning, send]);

  if (error) {
    return <ErrorCard error={error} reset={() => refetch()} />;
  }

  return (
    <div className="flex h-full flex-col justify-start gap-2 overflow-hidden">
      <div className="flex h-11 w-full flex-row items-center justify-between border-none px-4 pt-2">
        <div
          className="text-basis flex cursor-pointer items-center justify-start gap-2"
          onClick={() => setExpanded(!expanded)}
        >
          <RiArrowRightSLine
            className={`shrink-0 transition-transform duration-[250ms] ${
              expanded ? 'rotate-90' : ''
            }`}
          />
          {isPending ? (
            <SkeletonElement />
          ) : (
            <span className="text-basis text-sm font-normal">{trigger.eventName}</span>
          )}
        </div>

        <Button
          kind="primary"
          appearance="outlined"
          size="medium"
          iconSide="right"
          label="Invoke"
          loading={invokeLoading}
          disabled={invokeLoading}
          onClick={() => {
            setInvokeOpen(true);
          }}
        />

        {invokeError && <Alert severity="error">{invokeError.message}</Alert>}
        <InvokeModal
          doesFunctionAcceptPayload={true}
          isOpen={invokeOpen}
          onCancel={() => setInvokeOpen(false)}
          onConfirm={async ({ data, user }) => {
            const res = await invoke({
              functionSlug: slug || '',
              data,
              user,
            });

            if (res?.data) {
              setInvokeOpen(false);
              toast.success('Function invoked');
            }
          }}
        />
      </div>

      {expanded && (
        <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4 px-4">
          {type === 'EVENT' && (
            <>
              <ElementWrapper label="Event name">
                {isPending ? <SkeletonElement /> : <TextElement>{trigger.eventName}</TextElement>}
              </ElementWrapper>
              <ElementWrapper label="Event ID">
                {isPending ? <SkeletonElement /> : <IDElement>{trigger.IDs[0]}</IDElement>}
              </ElementWrapper>
              <ElementWrapper label="Received at">
                {isPending ? (
                  <SkeletonElement />
                ) : (
                  <TimeElement date={new Date(trigger.timestamp)} />
                )}
              </ElementWrapper>
            </>
          )}
          {type === 'CRON' && trigger?.cron && (
            <>
              <ElementWrapper label="Cron expression">
                {isPending ? <SkeletonElement /> : <CodeElement value={trigger.cron} />}
              </ElementWrapper>
              <ElementWrapper label="Cron ID">
                {isPending ? <SkeletonElement /> : <IDElement>{trigger.IDs[0]}</IDElement>}
              </ElementWrapper>
              <ElementWrapper label="Triggered at">
                {isPending ? (
                  <SkeletonElement />
                ) : (
                  <TimeElement date={new Date(trigger.timestamp)} />
                )}
              </ElementWrapper>
            </>
          )}
          {type === 'BATCH' && (
            <>
              <ElementWrapper label="Event name">
                {isPending ? (
                  <SkeletonElement />
                ) : (
                  <TextElement>{trigger.eventName ?? '-'}</TextElement>
                )}
              </ElementWrapper>
              <ElementWrapper label="Batch ID">
                {isPending ? <SkeletonElement /> : <IDElement>{trigger.batchID}</IDElement>}
              </ElementWrapper>
              <ElementWrapper label="Received at">
                {isPending ? (
                  <SkeletonElement />
                ) : (
                  <TimeElement date={new Date(trigger.timestamp)} />
                )}
              </ElementWrapper>
            </>
          )}
        </div>
      )}
      {result?.error && <ErrorInfo error={result.error.message || 'Unknown error'} />}
      <div className="flex-1">
        <Tabs
          defaultActive={result?.error ? 'error' : prettyPayload ? 'input' : 'output'}
          tabs={[
            ...(prettyPayload
              ? [
                  {
                    label: 'Input',
                    id: 'input',
                    node: (
                      <IO
                        title="Function Payload"
                        raw={prettyPayload}
                        actions={codeBlockActions}
                        loading={isPending || resultLoading}
                      />
                    ),
                  },
                ]
              : []),
            ...(prettyOutput
              ? [
                  {
                    label: 'Output',
                    id: 'output',
                    node: (
                      <IO title="Output" raw={prettyOutput} loading={isPending || resultLoading} />
                    ),
                  },
                ]
              : []),
            ...(result?.error
              ? [
                  {
                    label: 'Error details',
                    id: 'error',
                    node: (
                      <IO
                        title={result.error.message || 'Unknown error'}
                        raw={prettyErrorBody ?? ''}
                        error={true}
                        loading={isPending || resultLoading}
                      />
                    ),
                  },
                ]
              : []),
            ...(trace?.metadata?.length
              ? [
                  {
                    label: 'Metadata',
                    id: 'metadata',
                    node: <MetadataAttrs metadata={trace.metadata} />,
                  },
                ]
              : []),
          ]}
        />
      </div>
    </div>
  );
};
