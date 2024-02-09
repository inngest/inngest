type Props = {
  value: string;
};

export function InlineCode({ value }: Props) {
  return (
    <code className="inline-flex items-center rounded bg-slate-200 px-2 py-1 font-mono text-xs font-semibold leading-none">
      {value}
    </code>
  );
}
