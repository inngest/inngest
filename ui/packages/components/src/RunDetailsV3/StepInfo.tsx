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
import { isStepInfoInvoke, isStepInfoSleep, isStepInfoWait } from './types';
import { maybeBooleanToString, type StepInfoType } from './utils';

export const StepInfo = ({ selectedStep }: { selectedStep: StepInfoType }) => {
  const [expanded, setExpanded] = useState(true);
  const [rerunModalOpen, setRerunModalOpen] = useState(false);

  const { runID, result, trace, rerunFromStep, pathCreator } = selectedStep;

  const delayText = formatMilliseconds(
    (toMaybeDate(trace.startedAt) ?? new Date()).getTime() - new Date(trace.queuedAt).getTime()
  );

  let duration = 0;
  if (trace.childrenSpans && trace.childrenSpans.length > 0) {
    trace.childrenSpans.forEach((child, i) => {
      if (!child.startedAt) {
        return;
      }

      duration +=
        (toMaybeDate(child.endedAt) ?? new Date()).getTime() - new Date(child.startedAt).getTime();
    });
  } else if (trace.startedAt) {
    duration =
      (toMaybeDate(trace.endedAt) ?? new Date()).getTime() - new Date(trace.startedAt).getTime();
  }

  let durationText = '-';
  if (duration > 0) {
    durationText = formatMilliseconds(duration);
  }

  let stepKindInfo = null;

  if (isStepInfoInvoke(trace.stepInfo)) {
    const timeout = toMaybeDate(trace.stepInfo.timeout);
    stepKindInfo = (
      <>
        <ElementWrapper label="Run">
          {trace.stepInfo.runID ? (
            <LinkElement href={pathCreator.runPopout({ runID: trace.stepInfo.runID })}>
              {trace.stepInfo.runID}
            </LinkElement>
          ) : (
            '-'
          )}
        </ElementWrapper>
        <ElementWrapper label="Timeout">
          {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
        </ElementWrapper>
        <ElementWrapper label="Timed out">
          <TextElement>{maybeBooleanToString(trace.stepInfo.timedOut)}</TextElement>
        </ElementWrapper>
      </>
    );
  } else if (isStepInfoSleep(trace.stepInfo)) {
    const sleepUntil = toMaybeDate(trace.stepInfo.sleepUntil);
    stepKindInfo = (
      <ElementWrapper label="Sleep until">
        {sleepUntil ? <Time value={sleepUntil} /> : <TextElement>-</TextElement>}
      </ElementWrapper>
    );
  } else if (isStepInfoWait(trace.stepInfo)) {
    const timeout = toMaybeDate(trace.stepInfo.timeout);
    stepKindInfo = (
      <>
        <ElementWrapper label="Event name">
          <TextElement>{trace.stepInfo.eventName}</TextElement>
        </ElementWrapper>
        <ElementWrapper label="Timeout">
          {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
        </ElementWrapper>
        <ElementWrapper label="Timed out">
          <TextElement>{maybeBooleanToString(trace.stepInfo.timedOut)}</TextElement>
        </ElementWrapper>
        <ElementWrapper className="w-full" label="Match expression">
          {trace.stepInfo.expression ? (
            <CodeElement value={trace.stepInfo.expression} />
          ) : (
            <TextElement>-</TextElement>
          )}
        </ElementWrapper>
      </>
    );
  }

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
              rerunFromStep={rerunFromStep}
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
