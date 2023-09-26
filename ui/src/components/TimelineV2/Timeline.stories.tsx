import { useEffect, useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react';

import type { RunHistoryItem } from '@/store/generated';
import { HistoryParser, type HistoryNode } from './historyParser/index';
import { Timeline } from './index';
import cancelsData from './storyData/cancels.json';
import failsWithoutStepsData from './storyData/failsWithoutSteps.json';
import failsWithPrecedingStepData from './storyData/failsWithPrecedingStep.json';
import noStepsData from './storyData/noSteps.json';
import parallelStepsData from './storyData/parallelSteps.json';
import sleepsData from './storyData/sleeps.json';
import succeedsWith2StepsData from './storyData/succeedsWith2Steps.json';
import timesOutWaitingForEventData from './storyData/timesOutWaitingForEvent.json';
import waitsForEventData from './storyData/waitsForEvent.json';

type PropsAndCustomArgs = React.ComponentProps<typeof Timeline> & {
  _rawHistory: RunHistoryItem[];
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
    const [history, setHistory] = useState<Record<string, HistoryNode>>(args.history);

    useEffect(() => {
      const parser = new HistoryParser();
      for (let i = 0; i <= _rawHistoryFrame; i++) {
        parser.append(_rawHistory[i]);
      }
      setHistory(parser.history);
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

function createStory(rawHistory: unknown): Story {
  const raw = rawHistory as RunHistoryItem[];

  return {
    args: {
      _rawHistory: raw,
      _rawHistoryFrame: raw.length - 1,
      history: new HistoryParser(raw).history,
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
  };
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
