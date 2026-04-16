import { useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Select, type Option } from '@inngest/components/Select/Select';
import type { TimeRangePreset } from '@inngest/components/Experiments';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { cn } from '@inngest/components/utils/classNames';
import { RiAddLine, RiArrowDownSLine } from '@remixicon/react';

import { colorForMetric } from '@/lib/experiments/colors';

type Props = {
  preset: TimeRangePreset;
  onPresetChange: (p: TimeRangePreset) => void;
  selectedVariants: string[];
  onSelectedVariantsChange: (v: string[]) => void;
  availableVariants: string[];
};

const TIME_OPTIONS: { id: TimeRangePreset; name: string }[] = [
  { id: '24h', name: 'Last 24 hours' },
  { id: '7d', name: 'Last 7 days' },
  { id: '30d', name: 'Last 30 days' },
];

function VariantMultiSelect({
  availableVariants,
  selectedVariants,
  onSelectedVariantsChange,
}: {
  availableVariants: string[];
  selectedVariants: string[];
  onSelectedVariantsChange: (v: string[]) => void;
}) {
  const [draft, setDraft] = useState<string[]>(selectedVariants);
  const [search, setSearch] = useState('');
  const [open, setOpen] = useState(false);

  const filtered = useMemo(() => {
    if (!search) return availableVariants;
    const lower = search.toLowerCase();
    return availableVariants.filter((v) => v.toLowerCase().includes(lower));
  }, [availableVariants, search]);

  const allSelected = draft.length === 0;

  function toggle(name: string) {
    setDraft((prev) => {
      // If currently "all" (empty array), start with all except the toggled one
      if (prev.length === 0) {
        return availableVariants.filter((v) => v !== name);
      }
      if (prev.includes(name)) {
        const next = prev.filter((v) => v !== name);
        // If removing this means all are deselected, go back to "all"
        return next.length === 0 ? [] : next;
      }
      const next = [...prev, name];
      // If all variants are now selected, reset to "all"
      return next.length === availableVariants.length ? [] : next;
    });
  }

  function isChecked(name: string) {
    return allSelected || draft.includes(name);
  }

  const label =
    selectedVariants.length === 0
      ? 'All variants'
      : selectedVariants.length === 1
      ? selectedVariants[0]!
      : `${selectedVariants.length} variants`;

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) {
          setDraft(selectedVariants);
          setSearch('');
        }
      }}
    >
      <PopoverTrigger asChild>
        <button
          type="button"
          className="border-muted text-basis hover:bg-canvasSubtle inline-flex items-center gap-1 rounded-md border px-2 py-1 text-xs"
        >
          {label}
          <RiArrowDownSLine className="text-muted h-3.5 w-3.5" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-64 p-0">
        <div className="flex flex-col">
          {/* Search */}
          <div className="px-3 pb-1 pt-2">
            <input
              type="text"
              placeholder="Search variants"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="border-muted text-basis placeholder:text-disabled w-full rounded border px-2 py-1.5 text-sm outline-none focus:border-primary-moderate"
            />
          </div>

          {/* Options */}
          <div className="max-h-48 overflow-y-auto py-1">
            {filtered.map((name, i) => (
              <button
                key={name}
                type="button"
                className="hover:bg-canvasSubtle flex w-full items-center gap-2 px-4 py-1.5"
                onClick={() => toggle(name)}
              >
                <span
                  className={cn(
                    'flex h-4 w-4 shrink-0 items-center justify-center rounded-sm border',
                    isChecked(name)
                      ? 'border-primary-moderate bg-primary-moderate text-alwaysWhite'
                      : 'border-muted',
                  )}
                >
                  {isChecked(name) && (
                    <svg
                      width="10"
                      height="8"
                      viewBox="0 0 10 8"
                      fill="none"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <path
                        d="M1 4L3.5 6.5L9 1"
                        stroke="currentColor"
                        strokeWidth="1.5"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      />
                    </svg>
                  )}
                </span>
                <span
                  className="h-2.5 w-2.5 shrink-0 rounded-full"
                  style={{
                    backgroundColor: colorForMetric(
                      availableVariants.indexOf(name),
                    ),
                  }}
                />
                <span className="text-basis truncate text-xs">{name}</span>
              </button>
            ))}
          </div>

          {/* Footer */}
          <div className="border-subtle flex items-center justify-between border-t px-2 py-1.5">
            <Button
              kind="secondary"
              appearance="ghost"
              size="small"
              label="Reset"
              onClick={() => {
                setDraft([]);
              }}
            />
            <Button
              kind="primary"
              appearance="solid"
              size="small"
              label="Apply"
              onClick={() => {
                onSelectedVariantsChange(draft);
                setOpen(false);
              }}
            />
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function ExperimentDetailToolbar({
  preset,
  onPresetChange,
  selectedVariants,
  onSelectedVariantsChange,
  availableVariants,
}: Props) {
  const selectedTimeOption =
    TIME_OPTIONS.find((o) => o.id === preset) ?? TIME_OPTIONS[0]!;

  return (
    <div className="flex items-center justify-between gap-3">
      <div className="flex items-center gap-2">
        <Select
          label="Time range"
          isLabelVisible={false}
          value={selectedTimeOption}
          onChange={(opt: Option) => {
            onPresetChange(opt.id as TimeRangePreset);
          }}
          size="small"
        >
          <Select.Button size="small">{selectedTimeOption.name}</Select.Button>
          <Select.Options>
            {TIME_OPTIONS.map((opt) => (
              <Select.Option key={opt.id} option={opt}>
                {opt.name}
              </Select.Option>
            ))}
          </Select.Options>
        </Select>

        <VariantMultiSelect
          availableVariants={availableVariants}
          selectedVariants={selectedVariants}
          onSelectedVariantsChange={onSelectedVariantsChange}
        />
      </div>

      <Button
        kind="secondary"
        appearance="outlined"
        size="small"
        icon={<RiAddLine className="h-4 w-4" />}
        label="New visualization"
        disabled
      />
    </div>
  );
}
