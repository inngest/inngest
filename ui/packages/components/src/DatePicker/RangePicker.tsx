import { useEffect, useRef, useState, type ReactNode } from 'react';
import * as Tabs from '@radix-ui/react-tabs';
import { RiArrowRightSLine } from '@remixicon/react';
import { isBefore, type Duration } from 'date-fns';

import { Button } from '../Button';
import { Input } from '../Forms/Input';
import { Pill } from '../Pill';
import { Popover, PopoverContent, PopoverTrigger } from '../Popover';
import {
  DURATION_STRING_REGEX,
  durationToString,
  longDateFormat,
  parseDuration,
  subtractDuration,
} from '../utils/date';
import { DateInputButton, type DateButtonProps } from './DateButton';
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
export type RangeChangeProps = RelativeProps | AbsoluteProps;

type RangePickerProps = Omit<DateButtonProps, 'defaultValue' | 'onChange'> & {
  placeholder?: string;
  onChange: (args: RangeChangeProps) => void;
  defaultValue?: RangeChangeProps;
  upgradeCutoff?: Date;
  triggerComponent?: React.ComponentType<DateButtonProps>;
  allowFuture?: boolean;
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
  <div className="text-basis">{formatAbsolute(absoluteRange)}</div>
);

const RelativeDisplay = ({ duration }: { duration: string }) => (
  <span className="text-basis truncate">Last {duration}</span>
);

export const RangePicker = ({
  placeholder,
  onChange,
  defaultValue,
  upgradeCutoff,
  triggerComponent: TriggerComponent = DateInputButton,
  allowFuture = false,
  ...props
}: RangePickerProps) => {
  const getInitialDisplayValue = (defaultValue: RangeChangeProps | undefined): ReactNode => {
    if (defaultValue) {
      if (defaultValue.type === 'relative') {
        return <RelativeDisplay duration={durationToString(defaultValue.duration)} />;
      } else if (defaultValue.start && defaultValue.end) {
        return <AbsoluteDisplay absoluteRange={defaultValue} />;
      }
    }
    return null;
  };

  const durationRef = useRef<HTMLInputElement | null>(null);
  const [open, setOpen] = useState(false);
  const [durationError, setDurationError] = useState<string>('');
  const [absoluteRange, setAbsoluteRange] = useState<AbsoluteRange>();
  const [showAbsolute, setShowAbsolute] = useState<boolean>(
    () => defaultValue?.type === 'absolute'
  );
  const [tab, setTab] = useState('start');
  const [displayValue, setDisplayValue] = useState<ReactNode | null>(
    getInitialDisplayValue(defaultValue)
  );
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

    if (!allowFuture && absoluteRange?.end && isBefore(new Date(), absoluteRange.end)) {
      setEndError('End date is in the future');
      setEndValid(false);
    }
  };

  useEffect(() => {
    return () => {
      setShowAbsolute(false);
      setDurationError('');
    };
  }, []);

  useEffect(() => {
    validateRange();
  }, [absoluteRange]);

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
        <TriggerComponent {...props}>
          {displayValue ? (
            displayValue
          ) : (
            <span className="text-disabled">{placeholder ? placeholder : 'Select dates'}</span>
          )}
        </TriggerComponent>
      </PopoverTrigger>
      <PopoverContent align="start">
        <div className="bg-canvasBase flex flex-row">
          <div className={`${showAbsolute && 'min-h-[584px]'} border-muted w-[250px] border-r`}>
            <div className="px-3 py-2">
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
                  className={`text-basis hover:bg-canvasMuted flex flex-row items-center justify-between px-4 py-2 text-sm ${
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
                    <Pill kind="primary" appearance="outlined">
                      Upgrade Plan
                    </Pill>
                  )}
                </div>
              );
            })}
            <div className="flex flex-col">
              <div
                className={`border-muted text-basis hover:bg-canvasMuted flex cursor-pointer items-center justify-between border-t px-4 py-2 text-sm ${
                  showAbsolute && 'bg-canvasMuted'
                }`}
                onClick={() => {
                  setShowAbsolute(!showAbsolute);
                  setDurationError('');
                }}
              >
                Absolute range
                <RiArrowRightSLine className="text-subtle h-6 w-6" />
              </div>
              {showAbsolute && (
                <div className="text-muted px-4 py-2 text-sm">{formatAbsolute(absoluteRange)}</div>
              )}
            </div>
          </div>
          {showAbsolute && (
            <div className="bg-canvasBase flex w-[354px] flex-col">
              <Tabs.Root className="flex flex-col" value={tab} onValueChange={setTab}>
                <Tabs.List className="flex shrink-0 px-4 pt-4" aria-label={`Select ${tab} date`}>
                  <Tabs.Trigger
                    className="text-muted data-[state=active]:text-basis data-[state=active]:border-contrast flex flex-1 
                      cursor-pointer select-none items-center justify-center border-b-2 border-transparent px-5 pb-1
                      text-sm outline-none"
                    value="start"
                  >
                    Start
                  </Tabs.Trigger>
                  <Tabs.Trigger
                    className="text-muted data-[state=active]:text-basis data-[state=active]:border-contrast flex flex-1 
                      cursor-pointer select-none items-center justify-center border-b-2 border-transparent px-5 pb-1
                      text-sm outline-none"
                    value="end"
                  >
                    End
                  </Tabs.Trigger>
                </Tabs.List>
                <Tabs.Content
                  className="bg-canvasBase grow rounded-b-md outline-none"
                  value="start"
                >
                  <DateTimePicker
                    onChange={(start: Date | undefined) => {
                      if (start) {
                        setAbsoluteRange({
                          ...absoluteRange,
                          start,
                        });
                      }
                    }}
                    valid={startValid}
                    setValid={setStartValid}
                    defaultValue={
                      absoluteRange?.start ||
                      (defaultValue?.type === 'absolute' && defaultValue.start) ||
                      subtractDuration(new Date(), { days: 1 })
                    }
                  />
                  <div className="flex flex-col">
                    {startError && <p className="text-error mx-4 mt-1 text-sm">{startError}</p>}
                    <div className="flox-row flex justify-between p-4">
                      <Button label="Cancel" appearance="ghost" onClick={() => setOpen(false)} />
                      <Button
                        label="Next"
                        kind="primary"
                        disabled={!absoluteRange?.start || !startValid}
                        onClick={() => setTab('end')}
                      />
                    </div>
                  </div>
                </Tabs.Content>
                <Tabs.Content className="bg-canvasBase grow rounded-b-md outline-none" value="end">
                  <DateTimePicker
                    onChange={(end: Date | undefined) => {
                      if (end) {
                        setAbsoluteRange({
                          ...absoluteRange,
                          end,
                        });
                      }
                    }}
                    valid={endValid}
                    setValid={setEndValid}
                    defaultValue={
                      absoluteRange?.end ||
                      (defaultValue?.type === 'absolute' && defaultValue.end) ||
                      new Date()
                    }
                  />
                  <div className="flex flex-col">
                    {endError && <p className="text-error mx-4 mt-1 text-sm">{endError}</p>}
                    <div className="flox-row flex justify-between p-4">
                      <Button label="Cancel" appearance="ghost" onClick={() => setOpen(false)} />
                      <div className="flex flex-row">
                        <Button
                          label="Previous"
                          kind="primary"
                          appearance="outlined"
                          onClick={() => setTab('start')}
                          className="mr-2"
                        />
                        <Button
                          label="Apply"
                          kind="primary"
                          disabled={
                            !startValid || !endValid || !absoluteRange?.end || !absoluteRange?.start
                          }
                          onClick={() => {
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
