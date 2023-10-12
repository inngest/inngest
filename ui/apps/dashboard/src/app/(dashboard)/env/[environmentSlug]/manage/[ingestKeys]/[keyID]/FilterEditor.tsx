type FilterEditorProps = {
  filter: 'events' | 'IPs';
  list: 'allow' | 'deny';
  children: React.ReactNode;
};

export function FilterEditor({ filter, list, children }: FilterEditorProps) {
  return (
    <div className="bg-slate-910 flex h-full w-6/12 flex-col rounded-lg text-white">
      <header className="rounded-t-lg border-b border-slate-800 bg-slate-900 px-5 py-3">
        <h2 className="text-base font-medium">{filter === 'events' ? 'Events' : 'IP Addresses'}</h2>
        <p className="mt-0.5 text-sm font-light tracking-wide text-slate-300">
          Line-separated list of {filter} that are{' '}
          <b className="text-white">{list === 'allow' ? 'allowed' : 'denied'}</b>. Leave blank for
          no filter.
        </p>
      </header>
      <div className="flex-1 overflow-auto">
        <div className="px-4 font-mono text-sm font-light text-white">{children}</div>
      </div>
    </div>
  );
}
