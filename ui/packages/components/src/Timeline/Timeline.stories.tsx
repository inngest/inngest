import { useEffect, useState } from 'react';
import {
  HistoryParser,
  type HistoryNode,
  type RawHistoryItem,
} from '@inngest/components/utils/historyParser';
import cancelsData from '@inngest/components/utils/historyParser/testData/cancels.json';
import failsWithPrecedingStepData from '@inngest/components/utils/historyParser/testData/failsWithPrecedingStep.json';
import failsWithoutStepsData from '@inngest/components/utils/historyParser/testData/failsWithoutSteps.json';
import noStepsData from '@inngest/components/utils/historyParser/testData/noSteps.json';
import parallelStepsData from '@inngest/components/utils/historyParser/testData/parallelSteps.json';
import sleepsData from '@inngest/components/utils/historyParser/testData/sleeps.json';
import succeedsWith2StepsData from '@inngest/components/utils/historyParser/testData/succeedsWith2Steps.json';
import timesOutWaitingForEventData from '@inngest/components/utils/historyParser/testData/timesOutWaitingForEvent.json';
import waitsForEventData from '@inngest/components/utils/historyParser/testData/waitsForEvent.json';
import type { Meta, StoryObj } from '@storybook/react';

import { Timeline } from './index';

type PropsAndCustomArgs = React.ComponentProps<typeof Timeline> & {
  _rawHistory: RawHistoryItem[];
  _rawHistoryFrame: number;
};

const meta = {
  title: 'Components/Timeline',
  component: Timeline,
  argTypes: {
    history: {
      control: { disable: true },
    },
  },
  parameters: {
    layout: 'centered',
  },

  // This custom render function lets the story conditionally show the final or
  // animated timeline. If there isn't a delayMS, then the static final history
  // is used. If there is a delayMS, then we'll simulate each history item being
  // added one at a time.
  render: ({ _rawHistory, _rawHistoryFrame, ...args }) => {
    const [history, setHistory] = useState<HistoryParser>(args.history);

    useEffect(() => {
      const parser = new HistoryParser();
      for (let i = 0; i <= _rawHistoryFrame; i++) {
        // @ts-ignore
        parser.append(_rawHistory[i]);
      }
      setHistory(parser);
    }, [_rawHistoryFrame]);

    return (
      <div style={{ width: 600 }}>
        <Timeline {...args} history={history} />
      </div>
    );
  },
  tags: ['autodocs'],
} satisfies Meta<PropsAndCustomArgs>;

export default meta;

type Story = StoryObj<PropsAndCustomArgs>;

function createStory(rawHistory: unknown) {
  const raw = rawHistory as RawHistoryItem[];

  return {
    args: {
      _rawHistory: raw,
      _rawHistoryFrame: raw.length - 1,
      getResult: async () => JSON.stringify('fake'),
      history: new HistoryParser(raw),
    },
    argTypes: {
      _rawHistory: {
        table: {
          // Hide in UI.
          disable: true,
        },
      },
      _rawHistoryFrame: {
        control: { min: 0, max: raw.length - 1, step: 1, type: 'range' },
        description: 'Not a real prop. Only used for animating timeline in Storybook.',
      },
    },
  } satisfies Story;
}

export const cancels = createStory(cancelsData);
export const failsWithoutSteps = createStory(failsWithoutStepsData);
export const failsWithPrecedingStep = createStory(failsWithPrecedingStepData);
export const noSteps = createStory(noStepsData);
export const parallelSteps = createStory(parallelStepsData);
export const sleeps = createStory(sleepsData);
export const succeedsWith2Steps = createStory(succeedsWith2StepsData);
export const timesOutWaitingForEvent = createStory(timesOutWaitingForEventData);
export const waitsForEvent = createStory(waitsForEventData);
