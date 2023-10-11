import GroupButton from '@/components/GroupButton/GroupButton';

const functionStates = ['active', 'archived'] as const;
export type FunctionState = (typeof functionStates)[number];
function isFunctionState(value: string): value is FunctionState {
  return functionStates.includes(value as FunctionState);
}

type FunctionStateFilterProps = {
  handleClick: (state: FunctionState) => void;
  selectedOption: FunctionState;
};

export default function FunctionStateFilter({
  handleClick,
  selectedOption,
}: FunctionStateFilterProps) {
  return (
    <div className="flex bg-white px-4 py-1">
      <GroupButton
        title="Filter functions by state"
        options={[
          {
            name: 'Active',
            id: 'active',
            icon: <div className="mr-2 inline-block h-2.5 w-2.5 rounded-full bg-teal-500" />,
          },
          {
            name: 'Archived',
            id: 'archived',
            icon: <div className="mr-2 inline-block h-2.5 w-2.5 rounded-full bg-slate-300" />,
          },
        ]}
        handleClick={(id) => {
          if (!isFunctionState(id)) {
            throw new Error(`invalid function state: ${id}`);
          }

          handleClick(id);
        }}
        selectedOption={selectedOption}
      />
    </div>
  );
}
