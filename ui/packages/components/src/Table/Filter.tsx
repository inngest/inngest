import { Select, type Option } from '@inngest/components/Select/Select';
import { type VisibilityState } from '@tanstack/react-table';

type TableFilterProps = {
  options: Option[];
  columnVisibility?: VisibilityState;
  setColumnVisibility: (state: VisibilityState) => void;
};

export function TableFilter({ options, columnVisibility, setColumnVisibility }: TableFilterProps) {
  const selectedValues = options.filter(
    (option) => columnVisibility && columnVisibility[option.id] === true
  );

  return (
    <Select
      multiple
      value={selectedValues}
      onChange={(value: Option[]) => {
        const newColumnVisibility: VisibilityState = {};
        options.forEach((option) => {
          newColumnVisibility[option.id] = value.some((v) => v.id === option.id);
        });
        setColumnVisibility(newColumnVisibility);
      }}
      label="Table columns"
      isLabelVisible={false}
      size="small"
    >
      <Select.Button isLabelVisible={false} size="small">
        <div className="text-left">Table columns</div>
      </Select.Button>
      <Select.Options className="right-0">
        {options.map((option) => {
          return (
            <Select.CheckboxOption key={option.id} option={option}>
              <span className="flex items-center gap-1 lowercase">
                <label className="first-letter:capitalize">{option.name}</label>
              </span>
            </Select.CheckboxOption>
          );
        })}
      </Select.Options>
    </Select>
  );
}
