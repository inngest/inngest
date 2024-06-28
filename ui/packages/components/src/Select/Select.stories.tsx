import type { Meta, StoryObj } from '@storybook/react';

import { Select, SelectGroup } from './Select';

const meta = {
  title: 'Components/Select',
  component: Select,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    children: 'Select',
  },
} satisfies Meta<typeof Select>;

export default meta;

type Story = StoryObj<typeof Select>;

export const WithLabel: Story = {
  render: () => (
    <Select onChange={() => {}} label="Status">
      <Select.Button>Select</Select.Button>
      <Select.Options>
        <Select.Option option={{ id: 'option1', name: 'Option 1' }}>Option 1</Select.Option>
      </Select.Options>
    </Select>
  ),
};

export const WithoutLabel: Story = {
  render: () => (
    <Select onChange={() => {}} label="Status" isLabelVisible={false}>
      <Select.Button>Select</Select.Button>
      <Select.Options>
        <Select.Option option={{ id: 'option1', name: 'Option 1' }}>Option 1</Select.Option>
      </Select.Options>
    </Select>
  ),
};

export const WithOption: Story = {
  render: () => (
    <Select onChange={() => {}} label="Status" isLabelVisible={false}>
      <Select.Button>Select</Select.Button>
      <Select.Options>
        <Select.Option option={{ id: 'option1', name: 'Option 1' }}>Option 1</Select.Option>
      </Select.Options>
    </Select>
  ),
};

export const WithCheckboxOption: Story = {
  render: () => (
    <Select onChange={() => {}} label="Status" isLabelVisible={false}>
      <Select.Button>Select</Select.Button>
      <Select.Options>
        <Select.CheckboxOption option={{ id: 'option1', name: 'Option 1' }}>
          Option 1
        </Select.CheckboxOption>
      </Select.Options>
    </Select>
  ),
};

export const GroupOfSelects: Story = {
  render: () => (
    <SelectGroup>
      <Select onChange={() => {}} label="Status" isLabelVisible={false}>
        <Select.Button>Select</Select.Button>
        <Select.Options>
          <Select.CheckboxOption option={{ id: 'option1', name: 'Option 1' }}>
            Option 1
          </Select.CheckboxOption>
        </Select.Options>
      </Select>
      <Select onChange={() => {}} label="Status" isLabelVisible={false}>
        <Select.Button>Select</Select.Button>
        <Select.Options>
          <Select.CheckboxOption option={{ id: 'option1', name: 'Option 1' }}>
            Option 1
          </Select.CheckboxOption>
        </Select.Options>
      </Select>
    </SelectGroup>
  ),
};
