import { useCallback, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { RiArrowUpSLine } from '@remixicon/react';
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
import { InvokeModal } from '../InvokeButton';
// NOTE - This component should be a shared component as part of the design system.
// Until then, we re-use it from the RunDetailsV2 as these are part of the same parent UI.
import { ErrorCard } from '../RunDetailsV2/ErrorCard';
import { useInvokeRun } from '../Shared/useInvokeRun';
import { usePrettyJson } from '../hooks/usePrettyJson';
import { IconCloudArrowDown } from '../icons/CloudArrowDown';
import type { Result } from '../types/functionRun';
import { devServerURL, useDevServer } from '../utils/useDevServer';
import { Input } from './Input';
import { Output } from './Output';
import { Tabs } from './Tabs';

type TopInfoProps = {
  slug?: string;
  getTrigger: (runID: string) => Promise<Trigger>;
  result?: Result;
  runID: string;
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

export const TopInfo = ({ slug, getTrigger, runID, result }: TopInfoProps) => {
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
    <div className="flex h-full flex-col gap-2">
      <div className="flex h-11 w-full flex-row items-center justify-between border-none px-4">
        <div className="text-basis flex items-center justify-start gap-2">
          <RiArrowUpSLine
            className={`cursor-pointer transition-transform duration-500 ${
              expanded ? 'rotate-180' : ''
            }`}
            onClick={() => setExpanded(!expanded)}
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
        <dl className="flex flex-wrap gap-4 px-4">
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
        </dl>
      )}

      <Tabs
        defaultActive={0}
        tabs={[
          {
            label: 'Input',
            node: <Input title="Function Payload" raw={prettyPayload} actions={codeBlockActions} />,
          },
          { label: 'Output', node: <Output raw={prettyOutput} /> },
        ]}
      />
    </div>
  );
};
