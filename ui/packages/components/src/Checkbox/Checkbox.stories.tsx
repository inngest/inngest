import type { Meta, StoryObj } from '@storybook/react';

import { Checkbox, LabeledCheckbox } from './Checkbox';

const meta = {
  title: 'Components/Checkbox',
  component: Checkbox,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    children: 'Checkbox',
  },
} satisfies Meta<typeof Checkbox>;

export default meta;

type Story = StoryObj<typeof Checkbox>;

export const Default: Story = {
  args: {},
};

export const Checked: Story = {
  args: {
    checked: true,
  },
};

export const Disabled: Story = {
  args: {
    disabled: true,
  },
};

export const DefaultWithLabel: Story = {
  render: () => <LabeledCheckbox label="Title goes here" />,
};

export const DefaultWithLabelAndDescription: Story = {
  render: () => <LabeledCheckbox label="Title goes here" description="Description goes here" />,
};
