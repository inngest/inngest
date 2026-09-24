import { relativeTime } from '@inngest/components/utils/date';

export type ProgressiveSearchPhase = 'searching' | 'paused' | 'cancelled' | 'complete' | 'error';

type Props = {
  phase: ProgressiveSearchPhase;
  hasCompletedScanResponse: boolean;
  matchCount: number;
  searchedThrough?: Date;
  compact?: boolean;
};

export function ProgressiveSearchStatus({
  phase,
  hasCompletedScanResponse,
  matchCount,
  searchedThrough,
  compact = false,
}: Props) {
  const status =
    phase === 'searching'
      ? hasCompletedScanResponse
        ? 'Searching'
        : 'Searching…'
      : phase === 'complete'
      ? 'Search complete'
      : phase === 'error'
      ? 'Paused after an error'
      : phase === 'cancelled'
      ? 'Search cancelled'
      : 'Automatic search paused';
  const parts = [status];

  if (hasCompletedScanResponse) {
    const formattedCount = new Intl.NumberFormat().format(matchCount);
    parts.unshift(
      `${formattedCount} ${matchCount === 1 ? 'match' : 'matches'}${
        phase === 'complete' ? '' : ' so far'
      }`
    );
  }
  if (compact && hasCompletedScanResponse && phase === 'searching') {
    parts.pop();
  }
  if (!compact && searchedThrough) {
    parts.push(`through ${searchedThrough.toLocaleString()} (${relativeTime(searchedThrough)})`);
  }

  return <span>{parts.join(' · ')}</span>;
}
