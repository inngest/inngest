type Props = {
  label: string;
  justFinished: boolean;
};

export const TabLabelWithLoadedIndicator = ({ label, justFinished }: Props) => (
  <span className="inline-flex items-center gap-1.5">
    {label}
    {justFinished && (
      <span
        aria-hidden
        className="bg-status-completed h-1.5 w-1.5 shrink-0 rounded-full motion-safe:animate-pulse"
      />
    )}
  </span>
);
