type Props = {
  className?: string;
  detail: React.ReactNode;
  term: string;
};

export function Description({ className, detail, term }: Props) {
  return (
    <div className={className}>
      <dt className="text-subtle pb-2 text-sm">{term}</dt>
      <dd className="text-basis">{detail ?? ''}</dd>
    </div>
  );
}
