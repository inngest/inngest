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

it('clears the keyboard row indicator on pointer movement while search stays focused', async () => {
  const onFilterChange = vi.fn();
  render(
    <EntityFilter
      type="app"
      entities={options}
      selectedEntities={[]}
      onFilterChange={onFilterChange}
    />
  );
  await open('app All');
  const input = screen.getByPlaceholderText('Search for app');
  input.focus();
  fireEvent.keyDown(input, { key: 'ArrowDown' });
  const listbox = screen.getByRole('listbox');
  expect(listbox.getAttribute('data-keyboard-navigation')).toBe('true');

  const second = screen.getByRole('option', { name: 'Second option' });
  fireEvent.pointerEnter(second, { clientX: 10, clientY: 10 });
  fireEvent.pointerMove(second, { clientX: 20, clientY: 20 });
  // Mouse events also exercise Headless UI's pointer tracker in jsdom.
  fireEvent.mouseEnter(second, { clientX: 10, clientY: 10 });
  fireEvent.mouseMove(second, { clientX: 20, clientY: 20 });
  expect(second.getAttribute('data-headlessui-state')).toContain('active');
  expect(listbox.getAttribute('data-keyboard-navigation')).toBe('false');
  expect(document.activeElement).toBe(input);
  expect(onFilterChange).not.toHaveBeenCalled();

  fireEvent.keyDown(input, { key: 'ArrowUp' });
  expect(listbox.getAttribute('data-keyboard-navigation')).toBe('true');
  expect(
    screen.getByRole('option', { name: 'First option' }).getAttribute('data-headlessui-state')
  ).toContain('active');
  expect(onFilterChange).not.toHaveBeenCalled();
});

it('distinguishes hover from keyboard navigation in a non-searchable listbox', async () => {
  render(
    <Select label="Field" value={options[0]} onChange={vi.fn()}>
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
  const listbox = screen.getByRole('listbox');
  expect(listbox.getAttribute('data-keyboard-navigation')).toBe('true');
  fireEvent.pointerMove(screen.getByRole('option', { name: 'Second option' }));
  expect(listbox.getAttribute('data-keyboard-navigation')).toBe('false');
  fireEvent.keyDown(listbox, { key: 'ArrowDown' });
  expect(listbox.getAttribute('data-keyboard-navigation')).toBe('true');
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

it('keeps the whole select disabled until it becomes available', async () => {
  const onChange = vi.fn();
  const content = (disabled: boolean) => (
    <Select label="Function" disabled={disabled} onChange={onChange} isLabelVisible={false}>
      <Select.Button>Select function</Select.Button>
      <Select.Options>
        <Select.Option option={options[0]}>First option</Select.Option>
      </Select.Options>
    </Select>
  );
  const view = render(content(true));
  const button = screen.getByRole('button', { name: 'Function Select function' });
  expect(button.hasAttribute('disabled')).toBe(true);
  fireEvent.click(button);
  expect(screen.queryByRole('listbox')).toBeNull();
  expect(onChange).not.toHaveBeenCalled();
  view.rerender(content(false));
  expect(button.hasAttribute('disabled')).toBe(false);
  await open('Function Select function');
  fireEvent.click(screen.getByRole('option', { name: 'First option' }));
  expect(onChange).toHaveBeenCalledExactlyOnceWith(options[0]);
});
