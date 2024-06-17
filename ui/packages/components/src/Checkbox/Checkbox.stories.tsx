import type { Meta, StoryObj } from '@storybook/react';

import { Checkbox } from './Checkbox';

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
  render: () => (
    <Checkbox.Wrapper>
      <Checkbox />
      <Checkbox.Label>Title goes here</Checkbox.Label>
    </Checkbox.Wrapper>
  ),
};

export const DefaultWithLabelAndDescription: Story = {
  render: () => (
    <Checkbox.Wrapper>
      <Checkbox />
      <Checkbox.Label>
        Title goes here
        <Checkbox.Description>Description goes here</Checkbox.Description>
      </Checkbox.Label>
    </Checkbox.Wrapper>
  ),
};
