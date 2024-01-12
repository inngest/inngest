type Props = {
  className?: string;
  detail: React.ReactNode;
  term: string;
};

export function Description({ className, detail, term }: Props) {
  return (
    <div className={className}>
      <dt className="pb-2 text-sm text-slate-400">{term}</dt>
      <dd className="text-slate-800">{detail ?? ''}</dd>
    </div>
  );
}
