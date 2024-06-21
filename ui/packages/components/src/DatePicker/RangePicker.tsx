import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import * as Tabs from '@radix-ui/react-tabs';
import { isBefore, type Duration } from 'date-fns';

import { Badge } from '../Badge';
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
import { DateTimePicker } from './DateTimePicker';

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
  upgradeCutoff?: Date;
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
  upgradeCutoff,
  ...props
}: RangePickerProps) => {
  const durationRef = useRef<HTMLInputElement | null>(null);
  const [open, setOpen] = useState(false);
  const [durationError, setDurationError] = useState<string>('');
  const [absoluteRange, setAbsoluteRange] = useState<AbsoluteRange>();
  const [showAbsolute, setShowAbsolute] = useState<boolean>();
  const [tab, setTab] = useState('start');
  const [displayValue, setDisplayValue] = useState<ReactNode | null>(null);
  const [startValid, setStartValid] = useState(true);
  const [endValid, setEndValid] = useState(true);
  const [startError, setStartError] = useState('');
  const [endError, setEndError] = useState('');

  const validateDuration = (duration: string, upgradeCutoff?: Date): boolean => {
    if (!upgradeCutoff) {
      return true;
    }

    const request = subtractDuration(new Date(), parseDuration(duration));
    return !isBefore(request, upgradeCutoff);
  };

  const validateRange = () => {
    setStartValid(true);
    setStartError('');
    setEndValid(true);
    setEndError('');

    if (
      absoluteRange?.start &&
      absoluteRange?.end &&
      isBefore(absoluteRange?.end, absoluteRange.start)
    ) {
      setStartError('Start date is after end date');
      setEndError('End date is before start date');
      setStartValid(false);
      setEndValid(false);
    }

    if (upgradeCutoff && absoluteRange?.start && isBefore(absoluteRange.start, upgradeCutoff)) {
      setStartError('Please upgrade for increased history limits');
      setStartValid(false);
    }
  };

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

    if (!DURATION_STRING_REGEX.test(e.target.value)) {
      setDurationError('Invalid duration');
      return;
    }

    if (!validateDuration(e.target.value, upgradeCutoff)) {
      setDurationError('Upgrade plan');
      return;
    }

    setDisplayValue(<RelativeDisplay duration={e.target.value} />);
    onChange({ type: 'relative', duration: parseDuration(e.target.value) });
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen} modal={true}>
      <PopoverTrigger asChild>
        <DateInputButton {...props}>
          {displayValue ? (
            displayValue
          ) : (
            <span className="text-slate-500">{placeholder ? placeholder : 'Select dates'}</span>
          )}
        </DateInputButton>
      </PopoverTrigger>
      <PopoverContent>
        <div className="flex flex-row bg-white">
          <div className={`${showAbsolute && 'min-h-[589px]'} border-muted w-[250px] border-r`}>
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
            {Object.entries(RELATIVES).map(([k, v], i) => {
              const planValid = validateDuration(k, upgradeCutoff);
              return (
                <div
                  key={`duration-${i}`}
                  className={`flex flex-row items-center justify-between py-2 pl-6 pr-3 text-sm font-normal text-slate-700 hover:bg-blue-50 ${
                    planValid ? 'cursor-pointer' : 'cursor-not-allowed'
                  }`}
                  {...(planValid && {
                    onClick: () => {
                      setDisplayValue(<RelativeDisplay duration={k} />);
                      onChange({ type: 'relative', duration: parseDuration(k) });
                      setDurationError('');
                      setOpen(false);
                    },
                  })}
                >
                  {v}
                  {!planValid && (
                    <Badge
                      className="border-indigo-500 px-2 py-0.5 text-xs text-indigo-500"
                      kind="outlined"
                    >
                      Upgrade Plan
                    </Badge>
                  )}
                </div>
              );
            })}
            <div className="flex flex-col">
              <div
                className={`border-muted cursor-pointer border-t px-6 py-3.5 text-sm font-normal text-slate-700 hover:bg-blue-50 ${
                  showAbsolute && 'bg-blue-50'
                }`}
                onClick={() => {
                  setShowAbsolute(!showAbsolute);
                  setDurationError('');
                }}
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
                    className="flex h-11 flex-1 cursor-pointer select-none items-center 
                      justify-center bg-white px-5 text-sm text-slate-500 outline-none 
                      data-[state=active]:text-indigo-500 data-[state=active]:shadow-[inset_0_-1px_0_0,0_1px_0_0]"
                    value="start"
                  >
                    Start
                  </Tabs.Trigger>
                  <Tabs.Trigger
                    className="flex h-11 flex-1 cursor-pointer select-none items-center 
                      justify-center bg-white px-5 text-sm text-slate-500 outline-none 
                      data-[state=active]:text-indigo-500 data-[state=active]:shadow-[inset_0_-1px_0_0,0_1px_0_0]"
                    value="end"
                  >
                    End
                  </Tabs.Trigger>
                </Tabs.List>
                <Tabs.Content className="grow rounded-b-md bg-white outline-none" value="start">
                  <DateTimePicker
                    onChange={(start: Date | undefined) => {
                      if (start) {
                        setAbsoluteRange({
                          ...absoluteRange,
                          start,
                        });
                        validateRange();
                      }
                    }}
                    valid={startValid}
                    setValid={setStartValid}
                    defaultValue={
                      absoluteRange?.start ||
                      defaultStart ||
                      subtractDuration(new Date(), { days: 1 })
                    }
                  />
                  <div className="flex flex-col">
                    {startError && <p className="mx-4 mt-1 text-sm text-red-500">{startError}</p>}
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
                        disabled={!absoluteRange?.start || !startValid}
                        btnAction={() => setTab('end')}
                      />
                    </div>
                  </div>
                </Tabs.Content>
                <Tabs.Content className="grow rounded-b-md bg-white outline-none" value="end">
                  <DateTimePicker
                    onChange={(end: Date | undefined) => {
                      if (end) {
                        setAbsoluteRange({
                          ...absoluteRange,

                          end,
                        });
                        validateRange();
                      }
                    }}
                    valid={endValid}
                    setValid={setEndValid}
                    defaultValue={absoluteRange?.end || defaultEnd || new Date()}
                  />
                  <div className="flex flex-col">
                    {endError && <p className="mx-4 mt-1 text-sm text-red-500">{endError}</p>}
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
                          disabled={
                            !startValid || !endValid || !absoluteRange?.end || !absoluteRange?.start
                          }
                          btnAction={() => {
                            setDisplayValue(<AbsoluteDisplay absoluteRange={absoluteRange} />);
                            onChange({
                              type: 'absolute',
                              start: absoluteRange?.start!,
                              end: absoluteRange?.end!,
                            });
                            endValid && setOpen(false);
                          }}
                        />
                      </div>
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
