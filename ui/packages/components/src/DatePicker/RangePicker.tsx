import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import * as Tabs from '@radix-ui/react-tabs';
import { type Duration } from 'date-fns';

import { Button } from '../Button';
import { Input } from '../Forms/Input';
import { Popover, PopoverContent, PopoverTrigger } from '../Popover';
import {
  DURATION_STRING_REGEX,
  longDateFormat,
  parseDuration,
  subtractDuration,
} from '../utils/date';
import { DateInputButton, type DateInputButtonProps } from './DateInputButton';
import { DateTimePicker } from './DateTimePIcker';

type RelativeProps = {
  type: 'relative';
  duration: Duration;
};
type AbsoluteProps = {
  type: 'absolute';
  start: Date;
  end: Date;
};
type RangeChangeProps = RelativeProps | AbsoluteProps;

type RangePickerProps = Omit<DateInputButtonProps, 'defaultValue' | 'onChange'> & {
  placeholder?: string;
  onChange: (args: RangeChangeProps) => void;
  defaultStart?: Date;
  defaultEnd?: Date;
};

const RELATIVES = {
  '30s': 'Last 30 seconds',
  '1m': 'Last 1 minute',
  '10m': 'Last 10 minutes',
  '30m': 'Last 30 minutes',
  '45m': 'Last 45 minutes',
  '1h': 'Last 1 hour',
  '12h': 'Last 12 hours',
  '1d': 'Last 1 day',
  '2d': 'Last 2 days',
  '3d': 'Last 3 days',
  '7d': 'Last 7 days',
  '30d': 'Last 30 days',
};

export type AbsoluteRange = {
  start?: Date;
  end?: Date;
};

const formatAbsolute = (absoluteRange?: AbsoluteRange) => (
  <>
    {absoluteRange?.start
      ? absoluteRange.start.toLocaleDateString('en-us', longDateFormat)
      : 'mm/dd/yyyy, hh:mm:ss:mmm'}{' '}
    -{' '}
    {absoluteRange?.end
      ? absoluteRange.end.toLocaleDateString('en-us', longDateFormat)
      : 'mm/dd/yyyy, hh:mm:ss:mmm'}
  </>
);

const AbsoluteDisplay = ({ absoluteRange }: { absoluteRange?: AbsoluteRange }) => (
  <Tooltip>
    <TooltipTrigger>
      <div className="w-[180px] truncate text-slate-500">{formatAbsolute(absoluteRange)}</div>
    </TooltipTrigger>
    <TooltipContent className="whitespace-pre-line">{formatAbsolute(absoluteRange)}</TooltipContent>
  </Tooltip>
);

const RelativeDisplay = ({ duration }: { duration: string }) => (
  <span className="truncate text-slate-500">{duration}</span>
);

export const RangePicker = ({
  placeholder,
  onChange,
  defaultStart,
  defaultEnd,
  ...props
}: RangePickerProps) => {
  const durationRef = useRef<HTMLInputElement | null>(null);
  const [open, setOpen] = useState(false);
  const [durationError, setDurationError] = useState<string>('');
  const [absoluteRange, setAbsoluteRange] = useState<AbsoluteRange>();
  const [showAbsolute, setShowAbsolute] = useState<boolean>();
  const [tab, setTab] = useState('start');
  const [displayValue, setDisplayValue] = useState<ReactNode | null>(null);

  useEffect(() => {
    return () => {
      setShowAbsolute(false);
      setDurationError('');
    };
  }, []);

  const processDuration = (e: any) => {
    if (e.key !== 'Enter') {
      setDurationError('');
      return;
    }

    if (DURATION_STRING_REGEX.test(e.target.value)) {
      setDisplayValue(<RelativeDisplay duration={e.target.value} />);
      onChange({ type: 'relative', duration: parseDuration(e.target.value) });
      setOpen(false);
    } else {
      setDurationError('Invalid duration');
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <DateInputButton {...props}>
          {displayValue ? (
            displayValue
          ) : (
            <span className="text-slate-500">{placeholder ? placeholder : 'Date Range'}</span>
          )}
        </DateInputButton>
      </PopoverTrigger>
      <PopoverContent>
        <div className="flex flex-row bg-white">
          <div className={`${showAbsolute && 'min-h-[589px]'} w-[250px] border-r border-slate-300`}>
            <div className="m-2">
              <Input
                ref={durationRef}
                type="text"
                placeholder="Enter Relative Time (5s, 1m, 2d)"
                minLength={2}
                maxLength={64}
                error={durationError}
                onKeyDown={processDuration}
              />
            </div>
            {Object.entries(RELATIVES).map(([k, v], i) => (
              <div
                key={`duration-${i}`}
                className="cursor-pointer px-6 py-2 text-sm font-normal text-slate-700 hover:bg-blue-50"
                onClick={() => {
                  setDisplayValue(<RelativeDisplay duration={k} />);
                  onChange({ type: 'relative', duration: parseDuration(k) });
                  setDurationError('');
                  setOpen(false);
                }}
              >
                {v}
              </div>
            ))}
            <div className="flex flex-col">
              <div
                className={`cursor-pointer border-t border-slate-300 px-6 py-3.5 text-sm font-normal text-slate-700 hover:bg-blue-50 ${
                  showAbsolute && 'bg-blue-50'
                }`}
                onClick={() => setShowAbsolute(!showAbsolute)}
              >
                Absolute Range
              </div>
              {showAbsolute && (
                <div className="px-[22px] py-2 text-[13px] font-normal text-slate-500">
                  {formatAbsolute(absoluteRange)}
                </div>
              )}
            </div>
          </div>
          {showAbsolute && (
            <div className="flex w-[354px] flex-col">
              <Tabs.Root className="flex flex-col" value={tab} onValueChange={setTab}>
                <Tabs.List
                  className="border-mauve6 flex shrink-0 border-b"
                  aria-label="Manage your account"
                >
                  <Tabs.Trigger
                    className="flex h-[52px] flex-1 cursor-pointer select-none items-center 
                      justify-center bg-white px-5 text-sm text-slate-500 outline-none 
                      data-[state=active]:text-indigo-500 data-[state=active]:shadow-[inset_0_-1px_0_0,0_1px_0_0]"
                    value="start"
                  >
                    Start
                  </Tabs.Trigger>
                  <Tabs.Trigger
                    className="flex h-[52px] flex-1 cursor-pointer select-none items-center 
                      justify-center bg-white px-5 text-sm text-slate-500 outline-none 
                      data-[state=active]:text-indigo-500 data-[state=active]:shadow-[inset_0_-1px_0_0,0_1px_0_0]"
                    value="end"
                  >
                    End
                  </Tabs.Trigger>
                </Tabs.List>
                <Tabs.Content className="grow rounded-b-md bg-white outline-none" value="start">
                  <DateTimePicker
                    onChange={(start: Date | undefined) =>
                      start && setAbsoluteRange({ start, end: absoluteRange?.end })
                    }
                    defaultValue={
                      absoluteRange?.start ||
                      defaultStart ||
                      subtractDuration(new Date(), { days: 1 })
                    }
                  />
                  <div className="flox-row flex justify-between p-4">
                    <Button
                      size="small"
                      label="Cancel"
                      appearance="text"
                      btnAction={() => setOpen(false)}
                    />
                    <Button
                      size="small"
                      label="Next"
                      kind="primary"
                      disabled={!absoluteRange?.start}
                      btnAction={() => setTab('end')}
                    />
                  </div>
                </Tabs.Content>
                <Tabs.Content className="grow rounded-b-md bg-white outline-none" value="end">
                  <DateTimePicker
                    onChange={(end: Date | undefined) =>
                      end && setAbsoluteRange({ start: absoluteRange?.start, end })
                    }
                    defaultValue={absoluteRange?.end || defaultEnd || new Date()}
                  />
                  <div className="flox-row flex justify-between p-4">
                    <Button
                      size="small"
                      label="Cancel"
                      appearance="text"
                      btnAction={() => setOpen(false)}
                    />
                    <div className="flex flex-row">
                      <Button
                        size="small"
                        label="Previous"
                        kind="primary"
                        appearance="outlined"
                        btnAction={() => setTab('start')}
                        className="mr-2"
                      />
                      <Button
                        size="small"
                        label="Apply"
                        kind="primary"
                        disabled={!absoluteRange?.end || !absoluteRange?.start}
                        btnAction={() => {
                          setDisplayValue(<AbsoluteDisplay absoluteRange={absoluteRange} />);
                          onChange({
                            type: 'absolute',
                            start: absoluteRange?.start!,
                            end: absoluteRange?.end!,
                          });
                          setOpen(false);
                        }}
                      />
                    </div>
                  </div>
                </Tabs.Content>
              </Tabs.Root>
            </div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
};
