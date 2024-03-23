import React from 'react';
import { getLocalTimeZone, today } from '@internationalized/date';
import type { Meta } from '@storybook/react';

import { Calendar } from './Calendar';

const meta: Meta<typeof Calendar> = {
  title: 'Components/Calendar',
  component: Calendar,
  parameters: {
    layout: 'centered',
  },
  args: {
    minValue: today(getLocalTimeZone()).subtract({ days: 3 }),
    maxValue: today(getLocalTimeZone()),
  },
  tags: ['autodocs'],
};

export default meta;

export const Example = (args: any) => <Calendar aria-label="Date" {...args} />;
