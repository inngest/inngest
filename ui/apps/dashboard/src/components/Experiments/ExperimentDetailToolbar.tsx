import { useMemo } from 'react';
import { Button } from '@inngest/components/Button';
import { Select, type Option } from '@inngest/components/Select/Select';
import type { TimeRangePreset } from '@inngest/components/Experiments';
import { RiAddLine } from '@remixicon/react';

type Props = {
  preset: TimeRangePreset;
  onPresetChange: (p: TimeRangePreset) => void;
  variantFilter: string | null;
  onVariantFilterChange: (v: string | null) => void;
  availableVariants: string[];
};

const TIME_OPTIONS: { id: TimeRangePreset; name: string }[] = [
  { id: '24h', name: 'Last 24 hours' },
  { id: '7d', name: 'Last 7 days' },
  { id: '30d', name: 'Last 30 days' },
];

const ALL_VARIANTS_OPTION: Option = { id: '__all__', name: 'All variants' };

export function ExperimentDetailToolbar({
  preset,
  onPresetChange,
  variantFilter,
  onVariantFilterChange,
  availableVariants,
}: Props) {
  const selectedTimeOption =
    TIME_OPTIONS.find((o) => o.id === preset) ?? TIME_OPTIONS[0]!;

  const variantOptions: Option[] = useMemo(
    () => [
      ALL_VARIANTS_OPTION,
      ...availableVariants.map((v) => ({ id: v, name: v })),
    ],
    [availableVariants],
  );

  const selectedVariantOption =
    variantFilter != null
      ? variantOptions.find((o) => o.id === variantFilter) ??
        ALL_VARIANTS_OPTION
      : ALL_VARIANTS_OPTION;

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

        <Select
          label="Variant"
          isLabelVisible={false}
          value={selectedVariantOption}
          onChange={(opt: Option) => {
            onVariantFilterChange(opt.id === '__all__' ? null : opt.id);
          }}
          size="small"
        >
          <Select.Button size="small">
            {selectedVariantOption.name}
          </Select.Button>
          <Select.Options>
            {variantOptions.map((opt) => (
              <Select.Option key={opt.id} option={opt}>
                {opt.name}
              </Select.Option>
            ))}
          </Select.Options>
        </Select>
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
