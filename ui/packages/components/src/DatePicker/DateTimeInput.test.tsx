import { useState } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest';

import { DateTimeInput } from './DateTimeInput';
import { DateTimePicker } from './DateTimePicker';
import { RangePicker } from './RangePicker';

const initial = new Date(2024, 0, 31, 13, 20, 30, 400);
function setup(date = initial, is24HourFormat = true) {
  const onSelect = vi.fn();
  function Harness() {
    const [valid, setValid] = useState(true);
    return (
      <>
        <DateTimeInput
          selectedDateTime={date}
          onSelect={onSelect}
          is24HourFormat={is24HourFormat}
          valid={valid}
          setValid={setValid}
        />
        <output>{String(valid)}</output>
      </>
    );
  }
  render(<Harness />);
  onSelect.mockClear();
  return onSelect;
}
const change = (label: string, value: string) =>
  fireEvent.change(screen.getByLabelText(label), { target: { value } });
afterEach(cleanup);

describe('DateTimeInput calendar and time validity', () => {
  it('does not normalize January 31 into March when February is entered', () => {
    const onSelect = setup();
    change('Month', '02');
    expect(onSelect).not.toHaveBeenCalled();
    expect(screen.getByRole('status').textContent).toBe('false');
    change('Point in time (Minutes)', '21');
    expect(onSelect).not.toHaveBeenCalled();
    expect(screen.getByRole('status').textContent).toBe('false');
    change('Day', '29');
    expect(onSelect).toHaveBeenLastCalledWith(new Date(2024, 1, 29, 13, 21, 30, 400));
    expect(screen.getByRole('status').textContent).toBe('true');
  });
  it('rejects February 29 when changing to a non-leap year', () => {
    const onSelect = setup(new Date(2024, 1, 29, 13, 20, 30, 400));
    change('Year', '2025');
    expect(onSelect).not.toHaveBeenCalled();
    expect(screen.getByRole('status').textContent).toBe('false');
  });
  it.each([
    ['Month', '13'],
    ['Day', '32'],
    ['Year', '1969'],
    ['Point in time (Hours)', '24'],
    ['Point in time (Minutes)', '60'],
    ['Point in time (Seconds)', '60'],
    ['Point in time (Milliseconds)', '1000'],
  ])('does not emit out-of-range %s', (field, value) => {
    const onSelect = setup();
    change(field!, value!);
    expect(onSelect).not.toHaveBeenCalled();
    expect(screen.getByRole('status').textContent).toBe('false');
  });
  it('keeps a cleared time part invalid when another part changes', () => {
    const onSelect = setup();
    change('Point in time (Minutes)', '');
    change('Point in time (Seconds)', '31');
    expect(onSelect).not.toHaveBeenCalled();
    expect(screen.getByRole('status').textContent).toBe('false');
  });
  it.each([
    ['AM', 0],
    ['PM', 12],
  ])('preserves 12-hour %s conversion', (period, hour) => {
    const onSelect = setup(initial, false);
    change('Point in time (Hours)', '12');
    change('Point in time (Period)', String(period));
    expect(onSelect).toHaveBeenLastCalledWith(new Date(2024, 0, 31, Number(hour), 20, 30, 400));
  });
  it('uses the same numeric year for validation and leap-day construction', () => {
    const onSelect = setup(new Date(2024, 1, 29, 13, 20, 30, 400));
    change('Year', '2e3');
    expect(onSelect).toHaveBeenLastCalledWith(new Date(2000, 1, 29, 13, 20, 30, 400));
    expect(screen.getByRole('status').textContent).toBe('true');
    change('Day', '28');
    change('Day', '29');
    expect(onSelect).toHaveBeenLastCalledWith(new Date(2000, 1, 29, 13, 20, 30, 400));
    expect(screen.getByRole('status').textContent).toBe('true');
  });
  it('preserves valid ISO paste', () => {
    const onSelect = setup();
    fireEvent.paste(screen.getByLabelText('Month'), {
      clipboardData: { getData: () => '2024-02-29T10:11:12.123' },
    });
    expect(onSelect).toHaveBeenLastCalledWith(new Date(2024, 1, 29, 10, 11, 12, 123));
  });
});

it('does not move the real DateTimePicker selection to a normalized date', () => {
  const onChange = vi.fn();
  function Picker() {
    const [valid, setValid] = useState(true);
    return (
      <>
        <DateTimePicker
          defaultValue={initial}
          onChange={onChange}
          valid={valid}
          setValid={setValid}
        />
        <output>{String(valid)}</output>
      </>
    );
  }
  render(<Picker />);
  onChange.mockClear();
  change('Month', '02');
  expect(onChange).not.toHaveBeenCalled();
  expect(screen.getByRole('status').textContent).toBe('false');
  change('Day', '29');
  expect(onChange).toHaveBeenLastCalledWith(new Date(2024, 1, 29, 13, 20, 30, 400));
});

describe('RangePicker incomplete calendar fields', () => {
  // jsdom has no ResizeObserver; the real Radix popover uses it for positioning.
  beforeAll(() => {
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe() {}
        unobserve() {}
        disconnect() {}
      }
    );
  });
  afterAll(() => vi.unstubAllGlobals());
  const end = new Date(2024, 1, 2, 13, 20, 30, 400);
  const cases = [
    ['Month', '00', '01', '02'],
    ['Day', '00', '31', '02'],
    ['Year', '0000', '2024', '2024'],
  ];

  it.each(cases)('keeps Next disabled for zero %s after a time edit', (field, zero, repair) => {
    const onChange = vi.fn();
    render(
      <>
        <div id="modals" />
        <RangePicker defaultValue={{ type: 'absolute', start: initial, end }} onChange={onChange} />
      </>
    );
    fireEvent.click(screen.getAllByRole('button')[0]!);
    const next = screen.getByRole('button', { name: 'Next' }) as HTMLButtonElement;
    expect(next.disabled).toBe(false);
    change(field!, zero!);
    expect(next.disabled).toBe(true);
    change('Point in time (Minutes)', '21');
    expect(next.disabled).toBe(true);
    expect(onChange).not.toHaveBeenCalled();
    change(field!, repair!);
    expect(next.disabled).toBe(false);
  });

  it.each(cases)(
    'keeps Apply disabled for zero %s after a time edit',
    (field, zero, _startRepair, repair) => {
      const onChange = vi.fn();
      render(
        <>
          <div id="modals" />
          <RangePicker
            defaultValue={{ type: 'absolute', start: initial, end }}
            onChange={onChange}
          />
        </>
      );
      fireEvent.click(screen.getAllByRole('button')[0]!);
      fireEvent.click(screen.getByRole('button', { name: 'Next' }));
      const apply = screen.getByRole('button', { name: 'Apply' }) as HTMLButtonElement;
      expect(apply.disabled).toBe(false);
      change(field!, zero!);
      expect(apply.disabled).toBe(true);
      change('Point in time (Minutes)', '21');
      expect(apply.disabled).toBe(true);
      fireEvent.click(apply);
      expect(onChange).not.toHaveBeenCalled();
      change(field!, repair!);
      expect(apply.disabled).toBe(false);
      fireEvent.click(apply);
      expect(onChange).toHaveBeenLastCalledWith({
        type: 'absolute',
        start: initial,
        end: new Date(2024, 1, 2, 13, 21, 30, 400),
      });
    }
  );

  it.each(['Month', 'Day', 'Year'])(
    'keeps a cleared %s invalid after another field changes',
    (field) => {
      const onSelect = setup();
      change(field, '');
      change('Point in time (Minutes)', '21');
      expect(screen.getByRole('status').textContent).toBe('false');
      expect(onSelect).not.toHaveBeenCalled();
    }
  );
});
