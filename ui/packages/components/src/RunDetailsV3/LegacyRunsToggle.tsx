import { Pill } from '../Pill';
import { useLegacyTrace } from '../Shared/useLegacyTrace';
import { Switch } from '../Switch';

export const LegacyRunsToggle = ({ traceAIEnabled }: { traceAIEnabled: boolean }) => {
  const {
    enabled: legacyTraceEnabled,
    ready: legacyTraceReady,
    toggle: toggleLegacyTrace,
  } = useLegacyTrace();
  return (
    traceAIEnabled &&
    legacyTraceReady && (
      <div className="flex flex-row items-center justify-end gap-2">
        <Pill kind="info" appearance="solid" className="h-6">
          Beta feature
        </Pill>
        <span className="text-sm">New runs view</span>
        <Switch
          checked={!legacyTraceEnabled}
          className="cursor-pointer"
          onClick={() => toggleLegacyTrace()}
        />
      </div>
    )
  );
};
