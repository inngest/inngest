import type { Meta, StoryObj } from '@storybook/react';

import Input from './Input';

const meta = {
  title: 'Components/Input',
  component: Input,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    value: 'This is the input value'
  }
} satisfies Meta<typeof Input>;

export default meta;

type Story = StoryObj<typeof Input>;

export const Default: Story = {};

export const WithPlaceholder: Story = {
    args: {
      placeholder: 'This is the placeholder',
      value: '',
    },
  };

export const isInvalid: Story = {
    args: {
      isInvalid: true
    },
  };
