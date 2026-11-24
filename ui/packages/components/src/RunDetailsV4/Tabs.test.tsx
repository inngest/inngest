import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import { Tabs } from './Tabs';

const tabs = [
  { id: 'input', label: 'Input', node: <p>Input payload</p> },
  { id: 'output', label: 'Output', node: <p>Output payload</p> },
  { id: 'metadata', label: 'Metadata', node: <p>Run metadata</p> },
];
const label = 'Run data';

afterEach(cleanup);

describe('RunDetailsV4 Tabs', () => {
  it('links the selected tab to its panel and mounts only the active content', () => {
    render(<Tabs label={label} tabs={tabs} />);

    const list = screen.getByRole('tablist', { name: label });
    const input = within(list).getByRole('tab', { name: 'Input', selected: true });
    const panel = screen.getByRole('tabpanel', { name: 'Input' });
    expect(input.getAttribute('aria-controls')).toBe(panel.id);
    expect(panel.getAttribute('aria-labelledby')).toBe(input.id);
    expect(within(panel).getByText('Input payload')).toBeTruthy();
    expect(screen.queryByText('Output payload')).toBeNull();
    expect(screen.queryByText('Run metadata')).toBeNull();
  });

  it('moves focus and switches panels with the arrow, End, and Home keys', async () => {
    render(<Tabs label={label} tabs={tabs} />);

    const input = screen.getByRole('tab', { name: 'Input' });
    const output = screen.getByRole('tab', { name: 'Output' });
    const metadata = screen.getByRole('tab', { name: 'Metadata' });
    fireEvent.mouseDown(input, { button: 0 });
    input.focus();
    expect(output.tabIndex).toBe(-1);

    fireEvent.keyDown(input, { key: 'ArrowRight' });
    await waitFor(() => expect(document.activeElement).toBe(output));
    expect(screen.getByRole('tabpanel', { name: 'Output' })).toBeTruthy();
    expect(screen.queryByText('Input payload')).toBeNull();

    fireEvent.keyDown(output, { key: 'End' });
    await waitFor(() => expect(document.activeElement).toBe(metadata));
    expect(metadata.getAttribute('aria-selected')).toBe('true');

    fireEvent.keyDown(metadata, { key: 'Home' });
    await waitFor(() => expect(document.activeElement).toBe(input));
    expect(input.getAttribute('aria-selected')).toBe('true');

    fireEvent.keyDown(input, { key: 'ArrowLeft' });
    await waitFor(() => expect(document.activeElement).toBe(metadata));
    expect(metadata.getAttribute('aria-selected')).toBe('true');
  });

  it('keeps mouse selection and resets when the requested default changes', () => {
    const { rerender } = render(<Tabs defaultActive="output" label={label} tabs={tabs} />);
    expect(screen.getByRole('tabpanel', { name: 'Output' })).toBeTruthy();

    fireEvent.mouseDown(screen.getByRole('tab', { name: 'Metadata' }), { button: 0 });
    expect(screen.getByRole('tabpanel', { name: 'Metadata' })).toBeTruthy();

    rerender(<Tabs defaultActive="output" label={label} tabs={tabs} />);
    expect(screen.getByRole('tabpanel', { name: 'Metadata' })).toBeTruthy();

    rerender(<Tabs defaultActive="input" label={label} tabs={tabs} />);
    expect(screen.getByRole('tabpanel', { name: 'Input' })).toBeTruthy();
  });

  it('keeps a selected panel when late-arriving data removes the current tab', () => {
    const { rerender } = render(<Tabs defaultActive="output" label={label} tabs={tabs} />);

    rerender(<Tabs defaultActive="output" label={label} tabs={[tabs[0], tabs[2]]} />);
    expect(screen.getByRole('tab', { name: 'Input', selected: true })).toBeTruthy();
    expect(screen.getByRole('tabpanel', { name: 'Input' })).toBeTruthy();
  });
});
