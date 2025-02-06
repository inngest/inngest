import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiArrowUpSLine } from '@remixicon/react';

import { AITrace } from '../AI/AITrace';
import { parseAIOutput } from '../AI/utils';
import {
  CodeElement,
  ElementWrapper,
  LinkElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
import { RerunModal } from '../Rerun/RerunModal';
// NOTE - This component should be a shared component as part of the design system.
// Until then, we re-use it from the RunDetailsV2 as these are part of the same parent UI.
import { Time } from '../Time';
import { usePrettyJson } from '../hooks/usePrettyJson';
import { formatMilliseconds, toMaybeDate } from '../utils/date';
import { Input } from './Input';
import { Output } from './Output';
import { Tabs } from './Tabs';
import {
  isStepInfoInvoke,
  isStepInfoSleep,
  isStepInfoWait,
  type StepInfoInvoke,
  type StepInfoSleep,
  type StepInfoWait,
} from './types';
import { maybeBooleanToString, type PathCreator, type StepInfoType } from './utils';

type StepKindInfoProps = {
  stepInfo: StepInfoType['trace']['stepInfo'];
  pathCreator: StepInfoType['pathCreator'];
};

const InvokeInfo = ({
  stepInfo,
  pathCreator,
}: {
  stepInfo: StepInfoInvoke;
  pathCreator: PathCreator;
}) => {
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

const getStepKindInfo = (props: StepKindInfoProps): JSX.Element | null =>
  isStepInfoInvoke(props.stepInfo) ? (
    <InvokeInfo stepInfo={props.stepInfo} pathCreator={props.pathCreator} />
  ) : isStepInfoSleep(props.stepInfo) ? (
    <SleepInfo stepInfo={props.stepInfo} />
  ) : isStepInfoWait(props.stepInfo) ? (
    <WaitInfo stepInfo={props.stepInfo} />
  ) : null;

export const StepInfo = ({ selectedStep }: { selectedStep: StepInfoType }) => {
  const [expanded, setExpanded] = useState(true);
  const [rerunModalOpen, setRerunModalOpen] = useState(false);

  const { runID, result, trace, pathCreator } = selectedStep;

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
    pathCreator,
  });

  const aiOutput = result?.data ? parseAIOutput(result.data) : undefined;
  const prettyInput = usePrettyJson(result?.input ?? '') || (result?.input ?? '');
  const prettyOutput = usePrettyJson(result?.data ?? '') || (result?.data ?? '');

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

          <span className="text-basis text-sm font-normal">{trace.name}</span>
        </div>
        {runID && trace.stepID && (
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
              input={prettyInput}
            />
          </>
        )}
      </div>

      {expanded && (
        <dl className="flex flex-wrap gap-4 px-4">
          <ElementWrapper label="Queued at">
            <TimeElement date={new Date(trace.queuedAt)} />
          </ElementWrapper>

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

          <ElementWrapper label="Delay">
            <TextElement>{delayText}</TextElement>
          </ElementWrapper>

          <ElementWrapper label="Duration">
            <TextElement>{durationText}</TextElement>
          </ElementWrapper>

          {stepKindInfo}

          {aiOutput && <AITrace aiOutput={aiOutput} />}
        </dl>
      )}

      <Tabs
        defaultActive={0}
        tabs={[
          { label: 'Input', node: <Input title="Step Input" raw={prettyInput} /> },
          { label: 'Output', node: <Output raw={prettyOutput} /> },
        ]}
      />
    </div>
  );
};
