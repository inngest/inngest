import { useMemo, useState } from 'react';
import {
  SelectWithSearch,
  type Option,
} from '@inngest/components/Select/Select';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';

import { colorForVariant } from '@/lib/experiments/colors';

type Props = {
  range: RangeChangeProps;
  onRangeChange: (r: RangeChangeProps) => void;
  daysAgoMax: number;
  selectedVariants: string[];
  onSelectedVariantsChange: (v: string[]) => void;
  availableVariants: string[];
};

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
  range,
  onRangeChange,
  daysAgoMax,
  selectedVariants,
  onSelectedVariantsChange,
  availableVariants,
}: Props) {
  return (
    <div className="flex items-center gap-2">
      <TimeFilter
        defaultValue={range}
        daysAgoMax={daysAgoMax}
        minDuration={{ hours: 24 }}
        onDaysChange={onRangeChange}
      />

      <VariantMultiSelect
        availableVariants={availableVariants}
        selectedVariants={selectedVariants}
        onSelectedVariantsChange={onSelectedVariantsChange}
      />
    </div>
  );
}
