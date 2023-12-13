export function Labeled({
  className,
  label,
  value,
}: {
  className?: string;
  label: string;
  value: React.ReactNode | null | undefined;
}) {
  return (
    <label className={className}>
      <span className="text-xs text-slate-600">{label}</span>
      <div>{value ?? '-'}</div>
    </label>
  );
}
