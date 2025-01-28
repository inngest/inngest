'use client';

import { useCallback } from 'react';
import { ContentCard } from '@inngest/components/ContentCard';
import { RunStatusIcon } from '@inngest/components/FunctionRunStatusIcons';
import { MetadataGrid } from '@inngest/components/Metadata';
import { OutputCard } from '@inngest/components/OutputCard';
import { Pill } from '@inngest/components/Pill';
import { Timeline } from '@inngest/components/Timeline';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import type { FunctionVersion } from '@inngest/components/types/functionVersion';
import type { HistoryParser } from '@inngest/components/utils/historyParser';
import { RiTimeLine } from '@remixicon/react';

import { CancelRunButton } from '../CancelRunButton';
import { RerunButton } from '../RerunButton';
import { CancellationSummary } from './CancellationSummary';
import { SleepingSummary } from './SleepingSummary';
import { WaitingSummary } from './WaitingSummary';
import { renderRunMetadata } from './runMetadataRenderer';

type FuncProps = {
  cancelRun?: (runID: string) => Promise<unknown>;
  functionVersion?: Pick<FunctionVersion, 'url' | 'version'>;
  rerun?: () => Promise<unknown>;
};
type LoadingRun = {
  func?: Pick<Function, 'name' | 'triggers'>;
  loading: true;
  history?: undefined;
  run?: Pick<FunctionRun, 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'>;
  getHistoryItemOutput?: (historyItemID: string) => Promise<string | undefined>;
  navigateToRun?: undefined;
};

type WithRun = {
  func: Pick<Function, 'name' | 'triggers'>;
  loading?: false;
  history: HistoryParser;
  getHistoryItemOutput: (historyItemID: string) => Promise<string | undefined>;
  run: Pick<FunctionRun, 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'>;
  navigateToRun: React.ComponentProps<typeof Timeline>['navigateToRun'];
};

type Props = FuncProps & (WithRun | LoadingRun);

export function RunDetails(props: Props) {
  const {
    func,
    functionVersion,
    getHistoryItemOutput,
    history,
    rerun,
    run,
    navigateToRun,
    loading = false,
  } = props;

  const runID = run?.id;
  const cancelRun = useCallback(async () => {
    if (!props.cancelRun || !runID) {
      return;
    }
    await props.cancelRun(runID);
  }, [props.cancelRun, runID]);

  const firstTrigger = (func?.triggers && func.triggers[0]) ?? null;
  const cron = firstTrigger && firstTrigger.type === 'CRON';

  const metadataItems = renderRunMetadata({
    functionRun: run,
    functionVersion,
    history,
  });

  const isSuccess = run?.status === 'COMPLETED';

  return (
    <ContentCard
      active
      button={
        !loading && (
          <div className="flex gap-2">
            {cancelRun && (
              <CancelRunButton disabled={Boolean(run?.endedAt)} hasIcon onClick={cancelRun} />
            )}
            {rerun && <RerunButton onClick={rerun} />}
          </div>
        )
      }
      title={func?.name || '...'}
      icon={run?.status && <RunStatusIcon status={run?.status} className="h-5 w-5" />}
      type="run"
      badge={
        cron ? (
          <div className="py-2">
            <Pill kind="warning" appearance="outlined">
              <RiTimeLine className="h-4 w-4" />
              {firstTrigger.value}
            </Pill>
          </div>
        ) : null
      }
      metadata={
        <div className="pt-8">
          <MetadataGrid metadataItems={metadataItems} loading={loading} />
        </div>
      }
    >
      {run && history && getHistoryItemOutput && (
        <>
          <div className="px-5 pt-4">
            {run.status && run.endedAt && run.output && (
              <OutputCard content={run.output} isSuccess={isSuccess} />
            )}
            <CancellationSummary history={history} />
            <WaitingSummary history={history} />
            <SleepingSummary history={history} />
          </div>

          <hr className="border-muted mt-8" />
          <div className="px-5 pt-4">
            <h3 className="text-muted py-4 text-sm">Timeline</h3>
            <Timeline
              getOutput={getHistoryItemOutput}
              history={history}
              navigateToRun={navigateToRun}
            />
          </div>
        </>
      )}
    </ContentCard>
  );
}
