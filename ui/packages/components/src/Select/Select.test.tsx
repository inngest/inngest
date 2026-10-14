import { useState } from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';

import EntityFilter from '../Filter/EntityFilter';
import { TableFilter } from '../Table/Filter';
import { Select, type Option } from './Select';

afterEach(cleanup);

const options: Option[] = [
  { id: 'one', name: 'First option' },
  { id: 'two', name: 'Second option' },
];

async function open(label: string) {
  const button = screen.getByRole('button', { name: label });
  button.focus();
  fireEvent.keyDown(button, { key: 'ArrowDown' });
  await screen.findByRole('listbox');
}

it('commits a single option once and closes the list', async () => {
  const onChange = vi.fn();
  render(
    <Select label="Field" value={options[0]} onChange={onChange}>
      <Select.Button>Choose field</Select.Button>
      <Select.Options>
        {options.map((option) => (
          <Select.Option key={option.id} option={option}>
            {option.name}
          </Select.Option>
        ))}
      </Select.Options>
    </Select>
  );
  await open('Field Choose field');
  fireEvent.click(screen.getByRole('option', { name: 'Second option' }));
  expect(onChange).toHaveBeenCalledExactlyOnceWith(options[1]);
  await waitFor(() => expect(screen.queryByRole('listbox')).toBeNull());
});

it('keeps disabled single-select options from committing', async () => {
  const onChange = vi.fn();
  render(
    <Select label="Field" onChange={onChange}>
      <Select.Button>Choose field</Select.Button>
      <Select.Options>
        <Select.Option option={{ id: 'disabled', name: 'Unavailable', disabled: true }}>
          Unavailable
        </Select.Option>
      </Select.Options>
    </Select>
  );
  await open('Field Choose field');
  fireEvent.click(screen.getByRole('option', { name: 'Unavailable' }));
  expect(onChange).not.toHaveBeenCalled();
});

it('filters case-insensitively and keeps hidden selections until Apply', async () => {
  const onFilterChange = vi.fn();
  render(
    <EntityFilter
      type="app"
      entities={options}
      selectedEntities={['one']}
      onFilterChange={onFilterChange}
    />
  );
  await open('app First option');
  fireEvent.change(screen.getByPlaceholderText('Search for app'), { target: { value: 'SECOND' } });
  expect(screen.queryByRole('option', { name: 'First option' })).toBeNull();
  fireEvent.click(screen.getByRole('option', { name: 'Second option' }));
  expect(onFilterChange).not.toHaveBeenCalled();
  fireEvent.click(screen.getByText('Apply').closest('button')!);
  expect(onFilterChange).toHaveBeenCalledExactlyOnceWith(['one', 'two']);
  await waitFor(() => expect(screen.queryByRole('listbox')).toBeNull());
});

it('stages Reset and commits the empty selection only on Apply', async () => {
  const onFilterChange = vi.fn();
  render(
    <EntityFilter
      type="app"
      entities={options}
      selectedEntities={['one']}
      onFilterChange={onFilterChange}
    />
  );
  await open('app First option');
  expect(screen.getByText('Apply').closest('button')!.hasAttribute('disabled')).toBe(true);
  fireEvent.click(screen.getByText('Reset').closest('button')!);
  expect(onFilterChange).not.toHaveBeenCalled();
  fireEvent.click(screen.getByText('Apply').closest('button')!);
  expect(onFilterChange).toHaveBeenCalledExactlyOnceWith([]);
});

it('updates table visibility immediately while keeping the multi-select open', async () => {
  const onChange = vi.fn();
  function Table() {
    const [visibility, setVisibility] = useState({ one: true, two: false });
    return (
      <TableFilter
        options={options}
        columnVisibility={visibility}
        setColumnVisibility={(next) => {
          onChange(next);
          setVisibility(next as typeof visibility);
        }}
      />
    );
  }
  render(<Table />);
  await open('Table columns Table columns');
  fireEvent.click(screen.getByRole('option', { name: 'Second option' }));
  expect(onChange).toHaveBeenCalledExactlyOnceWith({ one: true, two: true });
  expect(screen.getByRole('listbox')).toBeTruthy();
  fireEvent.keyDown(screen.getByRole('listbox'), { key: 'Escape' });
  await waitFor(() => expect(screen.queryByRole('listbox')).toBeNull());
});
