import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiArrowRightSLine } from '@remixicon/react';

import { AITrace } from '../AI/AITrace';
import { parseAIOutput } from '../AI/utils';
import {
  CodeElement,
  ElementWrapper,
  LinkElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/NewElement';
import { RerunModal } from '../Rerun/RerunModal';
import { useShared } from '../SharedContext/SharedContext';
import { useGetTraceResult } from '../SharedContext/useGetTraceResult';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { Time } from '../Time';
import { usePrettyErrorBody, usePrettyJson, usePrettyShortError } from '../hooks/usePrettyJson';
import { formatMilliseconds, toMaybeDate } from '../utils/date';
import { ErrorInfo } from './ErrorInfo';
import { IO } from './IO';
import { Tabs } from './Tabs';
import { UserlandAttrs } from './UserlandAttrs';
import {
  isStepInfoInvoke,
  isStepInfoSignal,
  isStepInfoSleep,
  isStepInfoWait,
  type StepInfoInvoke,
  type StepInfoSignal,
  type StepInfoSleep,
  type StepInfoWait,
} from './types';
import { maybeBooleanToString, type StepInfoType } from './utils';

type StepKindInfoProps = {
  stepInfo: StepInfoType['trace']['stepInfo'];
};

const InvokeInfo = ({ stepInfo }: { stepInfo: StepInfoInvoke }) => {
  const { pathCreator } = usePathCreator();
  const timeout = toMaybeDate(stepInfo.timeout);
  return (
    <>
      <ElementWrapper label="Run">
        {stepInfo.runID ? (
          <LinkElement href={pathCreator.runPopout({ runID: stepInfo.runID })}>
            {stepInfo.runID}
          </LinkElement>
        ) : (
          '-'
        )}
      </ElementWrapper>
      <ElementWrapper label="Timeout">
        {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
      </ElementWrapper>
      <ElementWrapper label="Timed out">
        <TextElement>{maybeBooleanToString(stepInfo.timedOut)}</TextElement>
      </ElementWrapper>
    </>
  );
};

const SleepInfo = ({ stepInfo }: { stepInfo: StepInfoSleep }) => {
  const sleepUntil = toMaybeDate(stepInfo.sleepUntil);
  return (
    <ElementWrapper label="Sleep until">
      {sleepUntil ? <Time value={sleepUntil} /> : <TextElement>-</TextElement>}
    </ElementWrapper>
  );
};

const WaitInfo = ({ stepInfo }: { stepInfo: StepInfoWait }) => {
  const timeout = toMaybeDate(stepInfo.timeout);
  return (
    <>
      <ElementWrapper label="Event name">
        <TextElement>{stepInfo.eventName}</TextElement>
      </ElementWrapper>
      <ElementWrapper label="Timeout">
        {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
      </ElementWrapper>
      <ElementWrapper label="Timed out">
        <TextElement>{maybeBooleanToString(stepInfo.timedOut)}</TextElement>
      </ElementWrapper>
      <ElementWrapper className="w-full" label="Match expression">
        {stepInfo.expression ? (
          <CodeElement value={stepInfo.expression} />
        ) : (
          <TextElement>-</TextElement>
        )}
      </ElementWrapper>
    </>
  );
};

const SignalInfo = ({ stepInfo }: { stepInfo: StepInfoSignal }) => {
  const timeout = toMaybeDate(stepInfo.timeout);
  return (
    <>
      <ElementWrapper label="Signal name">
        <TextElement>{stepInfo.signal}</TextElement>
      </ElementWrapper>
      <ElementWrapper label="Timeout">
        {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
      </ElementWrapper>
      <ElementWrapper label="Timed out">
        <TextElement>{maybeBooleanToString(stepInfo.timedOut)}</TextElement>
      </ElementWrapper>
    </>
  );
};

const getStepKindInfo = (props: StepKindInfoProps): JSX.Element | null =>
  isStepInfoInvoke(props.stepInfo) ? (
    <InvokeInfo stepInfo={props.stepInfo} />
  ) : isStepInfoSleep(props.stepInfo) ? (
    <SleepInfo stepInfo={props.stepInfo} />
  ) : isStepInfoWait(props.stepInfo) ? (
    <WaitInfo stepInfo={props.stepInfo} />
  ) : isStepInfoSignal(props.stepInfo) ? (
    <SignalInfo stepInfo={props.stepInfo} />
  ) : null;

export const StepInfo = ({
  selectedStep,
  pollInterval: initialPollInterval,
  tracesPreviewEnabled,
}: {
  selectedStep: StepInfoType;

  pollInterval?: number;
  tracesPreviewEnabled?: boolean;
}) => {
  const { cloud } = useShared();
  const [expanded, setExpanded] = useState(true);
  const [rerunModalOpen, setRerunModalOpen] = useState(false);
  const { runID, trace } = selectedStep;
  const [pollInterval, setPollInterval] = useState(initialPollInterval);
  const { loading, data: result } = useGetTraceResult({
    traceID: trace.outputID,
    refetchInterval: pollInterval ? pollInterval : undefined,
    preview: tracesPreviewEnabled,
  });

  useEffect(() => {
    result && setPollInterval(undefined);
  }, [result]);

  const delayText = formatMilliseconds(
    (toMaybeDate(trace.startedAt) ?? new Date()).getTime() - new Date(trace.queuedAt).getTime()
  );

  const duration = trace.childrenSpans?.length
    ? trace.childrenSpans
        .filter((child) => child.startedAt)
        .reduce(
          (total, child) =>
            total +
            ((toMaybeDate(child.endedAt) ?? new Date()).getTime() -
              new Date(child.startedAt!).getTime()),
          0
        )
    : trace.startedAt
    ? (toMaybeDate(trace.endedAt) ?? new Date()).getTime() - new Date(trace.startedAt).getTime()
    : 0;

  const durationText = duration > 0 ? formatMilliseconds(duration) : '-';

  const stepKindInfo = getStepKindInfo({
    stepInfo: trace.stepInfo,
  });

  const aiOutput = result?.data ? parseAIOutput(result.data) : undefined;
  const prettyInput = usePrettyJson(result?.input ?? '') || (result?.input ?? '');
  const prettyOutput = usePrettyJson(result?.data ?? '') || (result?.data ?? '');
  const prettyErrorBody = usePrettyErrorBody(result?.error);
  const prettyShortError = usePrettyShortError(result?.error);

  return (
    <div className="flex h-full flex-col justify-start gap-2">
      <div className="flex min-h-11 w-full flex-row items-center justify-between border-none px-4">
        <div
          className="text-basis flex cursor-pointer items-center justify-start gap-2"
          onClick={() => setExpanded(!expanded)}
        >
          <RiArrowRightSLine
            className={`shrink-0 transition-transform duration-[250ms] ${
              expanded ? 'rotate-90' : ''
            }`}
          />

          <span className="text-basis text-sm font-normal">{trace.name}</span>
        </div>
        {runID && trace.stepID && (!cloud || prettyInput) && (
          <>
            <Button
              kind="primary"
              appearance="outlined"
              size="medium"
              label="Rerun from step"
              onClick={() => setRerunModalOpen(true)}
            />
            <RerunModal
              open={rerunModalOpen}
              setOpen={setRerunModalOpen}
              runID={runID}
              stepID={trace.stepID}
              input={prettyInput || result?.input || ''}
            />
          </>
        )}
      </div>

      {expanded && (
        <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4 px-4">
          {!trace.isUserland && (
            <ElementWrapper label="Queued at">
              <TimeElement date={new Date(trace.queuedAt)} />
            </ElementWrapper>
          )}

          <ElementWrapper label="Started at">
            {trace.startedAt ? (
              <TimeElement date={new Date(trace.startedAt)} />
            ) : (
              <TextElement>-</TextElement>
            )}
          </ElementWrapper>

          <ElementWrapper label="Ended at">
            {trace.endedAt ? (
              <TimeElement date={new Date(trace.endedAt)} />
            ) : (
              <TextElement>-</TextElement>
            )}
          </ElementWrapper>

          {!trace.isUserland && (
            <ElementWrapper label="Delay">
              <TextElement>{delayText}</TextElement>
            </ElementWrapper>
          )}

          <ElementWrapper label="Duration">
            <TextElement>{durationText}</TextElement>
          </ElementWrapper>

          {stepKindInfo}

          {aiOutput && <AITrace aiOutput={aiOutput} />}
        </div>
      )}

      {trace.isUserland && trace.userlandSpan ? (
        <UserlandAttrs userlandSpan={trace.userlandSpan} />
      ) : (
        <>
          {result?.error && <ErrorInfo error={prettyShortError} />}
          <div className="flex-1">
            <Tabs
              defaultActive={result?.error ? 'error' : 'output'}
              tabs={[
                ...(prettyInput
                  ? [
                      {
                        label: 'Input',
                        id: 'input',
                        node: <IO title="Step Input" raw={prettyInput} loading={loading} />,
                      },
                    ]
                  : []),
                ...(prettyOutput
                  ? [
                      {
                        label: 'Output',
                        id: 'output',
                        node: <IO title="Step Output" raw={prettyOutput} loading={loading} />,
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
                            title={prettyShortError}
                            raw={prettyErrorBody ?? ''}
                            error={true}
                            loading={loading}
                          />
                        ),
                      },
                    ]
                  : []),
              ]}
            />
          </div>
        </>
      )}
    </div>
  );
};
