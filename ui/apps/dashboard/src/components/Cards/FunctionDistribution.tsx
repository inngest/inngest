interface Props {
  activeCount: number;
  disabledCount: number;
  removedCount: number;
}

export function FunctionDistribution({ activeCount, disabledCount, removedCount }: Props) {
  const totalCount = activeCount + disabledCount + removedCount;
  const activePercent = Math.round((activeCount / totalCount) * 100);
  const disabledPercent = Math.round((disabledCount / totalCount) * 100);
  const removedPercent = Math.round((removedCount / totalCount) * 100);

  return (
    <div className="mt-4 flex items-center gap-2">
      {activeCount > 0 && (
        <div style={{ flexBasis: `${activePercent}%` }}>
          <span className="block h-3 rounded bg-teal-400" />
          <span className="mt-2 block whitespace-nowrap text-center text-xs text-slate-500">
            {activeCount} Active
          </span>
        </div>
      )}

      {disabledCount > 0 && (
        <div style={{ flexBasis: `${disabledPercent}%` }}>
          <span className="block h-3 rounded bg-slate-600" />
          <span className="mt-2 block whitespace-nowrap text-center text-xs text-slate-500">
            {disabledCount} Disabled
          </span>
        </div>
      )}

      {removedCount > 0 && (
        <div style={{ flexBasis: `${removedPercent}%` }}>
          <span className="block h-3 rounded bg-red-400" />
          <span className="mt-2 block whitespace-nowrap text-center text-xs text-slate-500">
            {removedCount} Removed
          </span>
        </div>
      )}
    </div>
  );
}
