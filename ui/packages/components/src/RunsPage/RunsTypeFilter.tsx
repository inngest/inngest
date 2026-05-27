import { Switch, SwitchLabel, SwitchWrapper } from '../Switch';

type RunsTypeFilterProps = {
  excludeDeferred: boolean;
  onExcludeDeferredChange: (value: boolean) => void;
};

// A simple on/off toggle: off shows all runs (no run-type filter), on excludes
// deferred runs by filtering to PRIMARY runs on the backend.
export default function RunsTypeFilter({
  excludeDeferred,
  onExcludeDeferredChange,
}: RunsTypeFilterProps) {
  return (
    <SwitchWrapper>
      <Switch
        id="exclude-deferred-runs"
        checked={excludeDeferred}
        onCheckedChange={onExcludeDeferredChange}
      />
      <SwitchLabel htmlFor="exclude-deferred-runs" className="text-sm">
        Exclude deferred runs
      </SwitchLabel>
    </SwitchWrapper>
  );
}
