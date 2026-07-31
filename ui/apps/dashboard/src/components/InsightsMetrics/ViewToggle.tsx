export type ViewMode = 'chart' | 'table';

// ViewToggle is a small chart/table mode switch for a card that can render
// its data either way (e.g. a range plot vs. the same rows as a table) —
// generic over the two modes' labels so it isn't tied to "chart"/"table"
// specifically. Same underline-tab row (full-width, bottom-bordered) as
// Scores/ScoreCard's AggregationTabs/TabRow, reused here rather than
// duplicated — meant to render as its own row between a card's title and
// its content, not inline in the header.
export function ViewToggle({
  mode,
  onChange,
  options = [
    { value: 'chart', label: 'Chart' },
    { value: 'table', label: 'Table' },
  ],
}: {
  mode: ViewMode;
  onChange: (mode: ViewMode) => void;
  options?: { value: ViewMode; label: string }[];
}) {
  return (
    <div className="border-subtle mb-3 flex flex-row gap-4 border-b">
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          onClick={() => onChange(option.value)}
          className={`-mb-px pb-1.5 text-sm ${
            mode === option.value
              ? 'text-basis border-contrast border-b-2 font-medium'
              : 'text-muted hover:text-basis'
          }`}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}
