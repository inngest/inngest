type Props = {
  value: string;
};

export function InlineCode({ value }: Props) {
  return (
    <code className="text-2xs inline-flex items-center rounded bg-slate-200 px-2 py-1 font-mono font-semibold leading-none">
      {value}
    </code>
  );
}
