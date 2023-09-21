import { useEffect, useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react';

import type { RunHistoryItem } from '@/store/generated';
import { HistoryParser, type HistoryNode } from './historyParser/historyParser';
import { Timeline } from './index';
import succeedsWith2StepsData from './storyData/succeedsWith2Steps.json';
import waitsForEventData from './storyData/waitsForEvent.json';

type PropsAndCustomArgs = React.ComponentProps<typeof Timeline> & {
  _delayMS?: number;
  _rawHistory: RunHistoryItem[];
};

const meta = {
  title: 'Components/Timeline',
  component: Timeline,
  argTypes: {
    _delayMS: {
      control: { type: 'range', min: 0, max: 5000, step: 100 },
      description: 'Not a real prop. Only used for animating timeline in Storybook.',
      defaultValue: 0,
    },
    _rawHistory: {
      table: {
        // Hide in UI.
        disable: true,
      },
    },

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
  render: ({ _delayMS, _rawHistory, ...args }) => {
    let defaultHistory = args.history;
    if (_delayMS) {
      defaultHistory = {};
    }

    const [history, setHistory] = useState<Record<string, HistoryNode>>(defaultHistory);

    useEffect(() => {
      if (!_delayMS) {
        return;
      }

      const parser = new HistoryParser();
      let i = 0;

      const timer = setInterval(() => {
        if (i > _rawHistory.length - 1) {
          return;
        }

        parser.append(_rawHistory[i]);
        setHistory(parser.history);
        i++;
      }, _delayMS);

      return () => clearInterval(timer);
    }, [_delayMS]);
    return <Timeline {...args} history={history} />;
  },
  tags: ['autodocs'],
} satisfies Meta<PropsAndCustomArgs>;

export default meta;

type Story = StoryObj<PropsAndCustomArgs>;

function createStory(rawHistory: unknown): Story {
  return {
    args: {
      _rawHistory: rawHistory as RunHistoryItem[],
      history: new HistoryParser(rawHistory as RunHistoryItem[]).history,
    },
  };
}

export const succeedsWith2Steps = createStory(succeedsWith2StepsData);
export const waitsForEvent = createStory(waitsForEventData);
