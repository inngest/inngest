import { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react';

import { Select, SelectGroup, SelectWithSearch, type Option } from './Select';

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

const options: Option[] = [
  {
    id: 'id1',
    name: 'Option 1',
  },
  {
    id: 'id2',
    name: 'Option 2',
  },
];

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
    <Select onChange={() => {}} label="Status" isLabelVisible={false} multiple>
      <Select.Button>Select</Select.Button>
      <Select.Options>
        <Select.CheckboxOption option={{ id: 'option1', name: 'Option 1' }}>
          Option 1
        </Select.CheckboxOption>
      </Select.Options>
    </Select>
  ),
};

export const SelectWithSearchInput: Story = {
  render: () => {
    const [selectedOption, setSelectedOption] = useState(options);
    const [query, setQuery] = useState('');

    const filteredOptions =
      query === ''
        ? options
        : options.filter((option) => {
            return option.name.toLowerCase().includes(query.toLowerCase());
          });
    return (
      <SelectWithSearch
        value={selectedOption}
        onChange={(value: Option[]) => {
          const newValue: Option[] = [];
          value.forEach((option) => {
            newValue.push(option);
          });
          setSelectedOption(newValue);
        }}
        label="Status"
        isLabelVisible={false}
        multiple
      >
        <SelectWithSearch.Button>Select Options</SelectWithSearch.Button>
        <SelectWithSearch.Options>
          <SelectWithSearch.SearchInput
            displayValue={(option: Option) => option?.name}
            placeholder="Search for option"
            onChange={(event) => setQuery(event.target.value)}
          />
          {filteredOptions.map((option) => (
            <SelectWithSearch.CheckboxOption key={option.id} option={option}>
              {option.name}
            </SelectWithSearch.CheckboxOption>
          ))}
          <SelectWithSearch.Footer onReset={() => setSelectedOption([])} />
        </SelectWithSearch.Options>
      </SelectWithSearch>
    );
  },
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
