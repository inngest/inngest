import RelativeTimeFilter from '@inngest/components/Filter/RelativeTimeFilter';
import { type Option } from '@inngest/components/Select/Select';

const daysAgoArray = ['1', '3', '7', '14', '30'];

type Props = {
  daysAgoMax: number;
  onDaysChange: (value: string) => void;
  selectedDays: string;
};

export function TimeFilter({ daysAgoMax, onDaysChange, selectedDays }: Props) {
  const options: Option[] = daysAgoArray.map((date) => ({
    id: date,
    name: date === '1' ? `Last ${date} day` : `Last ${date} days`,
    disabled: parseInt(date) > daysAgoMax,
  }));

  const selectedValue = options.find((option) => option.id === selectedDays.toString());

  /* TODO: better plan validation and toast when absolute time available */
  // If selected value is disabled, select 3 days instead
  if (selectedValue && selectedValue.disabled) {
    const newSelectedValue = options.find(
      (option) => !option.disabled && parseInt(option.id) === 3
    );
    if (newSelectedValue) {
      onDaysChange(newSelectedValue.id);
    }
  }

  return (
    <RelativeTimeFilter
      options={options}
      selectedDays={selectedValue}
      onDaysChange={(value: Option) => {
        const numericId = parseInt(value.id);
        if (!isNaN(numericId)) {
          onDaysChange(value.id);
        }
      }}
    />
  );
}
