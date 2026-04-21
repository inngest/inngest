import { useMemo, useState } from 'react';
import {
  Select,
  SelectWithSearch,
  type Option,
} from '@inngest/components/Select/Select';
import type { TimeRangePreset } from '@inngest/components/Experiments';

import { colorForVariant } from '@/lib/experiments/colors';

type Props = {
  preset: TimeRangePreset;
  onPresetChange: (p: TimeRangePreset) => void;
  selectedVariants: string[];
  onSelectedVariantsChange: (v: string[]) => void;
  availableVariants: string[];
};

const TIME_OPTIONS = [
  { id: '24h', name: 'Last 24 hours' },
  { id: '7d', name: 'Last 7 days' },
  { id: '30d', name: 'Last 30 days' },
] as const satisfies readonly { id: TimeRangePreset; name: string }[];

const DEFAULT_TIME_OPTION = TIME_OPTIONS[0];

const TIME_PRESET_SET = new Set<string>(TIME_OPTIONS.map((o) => o.id));

function isTimeRangePreset(id: string): id is TimeRangePreset {
  return TIME_PRESET_SET.has(id);
}

function VariantMultiSelect({
  availableVariants,
  selectedVariants,
  onSelectedVariantsChange,
}: {
  availableVariants: string[];
  selectedVariants: string[];
  onSelectedVariantsChange: (v: string[]) => void;
}) {
  const [query, setQuery] = useState('');

  const options: Option[] = useMemo(
    () => availableVariants.map((name) => ({ id: name, name })),
    [availableVariants],
  );

  const selectedOptions = useMemo(
    () => options.filter((o) => selectedVariants.includes(o.id)),
    [options, selectedVariants],
  );

  const filteredOptions = useMemo(() => {
    if (!query) return options;
    const lower = query.toLowerCase();
    return options.filter((o) => o.name.toLowerCase().includes(lower));
  }, [options, query]);

  const [firstSelected] = selectedVariants;
  const label =
    selectedVariants.length === 0 || firstSelected === undefined
      ? 'All variants'
      : selectedVariants.length === 1
      ? firstSelected
      : `${selectedVariants.length} variants`;

  const handleChange = (value: Option[]) => {
    const next = value.map((o) => o.id);
    // "Empty = all" sentinel: collapse full selection to empty.
    const collapsed = next.length === availableVariants.length ? [] : next;
    onSelectedVariantsChange(collapsed);
  };

  const handleReset = () => {
    onSelectedVariantsChange([]);
  };

  return (
    <SelectWithSearch
      multiple
      value={selectedOptions}
      onChange={handleChange}
      label="Variants"
      isLabelVisible={false}
      size="small"
    >
      <SelectWithSearch.Button size="small">
        <span className="text-basis text-xs">{label}</span>
      </SelectWithSearch.Button>
      <SelectWithSearch.Options className="w-64">
        <SelectWithSearch.SearchInput
          displayValue={() => ''}
          placeholder="Search variants"
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            setQuery(e.target.value)
          }
        />
        <div className="max-h-48 overflow-y-auto">
          {filteredOptions.map((opt) => (
            <SelectWithSearch.CheckboxOption key={opt.id} option={opt}>
              <span className="inline-flex items-center gap-2">
                <span
                  className="h-2.5 w-2.5 shrink-0 rounded-full"
                  style={{
                    backgroundColor: colorForVariant(
                      availableVariants.indexOf(opt.id),
                    ),
                  }}
                />
                <span className="text-basis truncate text-xs">{opt.name}</span>
              </span>
            </SelectWithSearch.CheckboxOption>
          ))}
        </div>
        <SelectWithSearch.Footer onReset={handleReset} />
      </SelectWithSearch.Options>
    </SelectWithSearch>
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
    TIME_OPTIONS.find((o) => o.id === preset) ?? DEFAULT_TIME_OPTION;

  return (
    <div className="flex items-center gap-2">
      <Select
        label="Time range"
        isLabelVisible={false}
        value={selectedTimeOption}
        onChange={(opt: Option) => {
          if (isTimeRangePreset(opt.id)) onPresetChange(opt.id);
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
  );
}
