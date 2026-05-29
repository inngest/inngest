import { Select, type Option } from '../Select/Select';

type RunsTypeFilterProps = {
  isDeferred: boolean | undefined;
  onChange: (value: boolean | undefined) => void;
};

const ALL_ID = 'ALL';
const PRIMARY_ID = 'PRIMARY';
const DEFERRED_ID = 'DEFERRED';

const options: Option[] = [
  { id: ALL_ID, name: 'All' },
  { id: PRIMARY_ID, name: 'Primary' },
  { id: DEFERRED_ID, name: 'Deferred' },
];

function idForFilter(isDeferred: boolean | undefined): string {
  if (isDeferred === undefined) return ALL_ID;
  return isDeferred ? DEFERRED_ID : PRIMARY_ID;
}

function filterForId(id: unknown): boolean | undefined {
  if (id === DEFERRED_ID) return true;
  if (id === PRIMARY_ID) return false;
  return undefined;
}

export default function RunsTypeFilter({ isDeferred, onChange }: RunsTypeFilterProps) {
  const selectedValue =
    options.find((option) => option.id === idForFilter(isDeferred)) ?? options[0];

  return (
    <Select
      value={selectedValue}
      onChange={(value: Option) => {
        onChange(filterForId(value.id));
      }}
      label="Type"
      isLabelVisible
      className="bg-modalBase min-w-[90px]"
      size="small"
    >
      <Select.Button isLabelVisible size="small">
        <span>{selectedValue?.name}</span>
      </Select.Button>
      <Select.Options>
        {options.map((option) => {
          return (
            <Select.Option key={option.id} option={option}>
              <span className="inline-flex items-center gap-2">
                <span>{option.name}</span>
              </span>
            </Select.Option>
          );
        })}
      </Select.Options>
    </Select>
  );
}
