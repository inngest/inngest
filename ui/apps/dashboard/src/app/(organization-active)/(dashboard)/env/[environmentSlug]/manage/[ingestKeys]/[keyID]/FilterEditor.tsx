type FilterEditorProps = {
  filter: 'events' | 'IPs';
  list: 'allow' | 'deny';
  children: React.ReactNode;
};

export function FilterEditor({ filter, list, children }: FilterEditorProps) {
  return (
    <div className="bg-canvasBase text-basis border-muted flex h-full w-6/12 flex-col rounded-md border text-sm">
      <header className="border-muted rounded-t-lg border-b px-5 py-3">
        <h2 className="text-base font-medium">{filter === 'events' ? 'Events' : 'IP Addresses'}</h2>
        <p className="text-subtle mt-0.5 text-sm font-light tracking-wide">
          Line-separated list of {filter} that are{' '}
          <b className="text-basis">{list === 'allow' ? 'allowed' : 'denied'}</b>. Leave blank for
          no filter.
        </p>
      </header>
      <div className="bg-codeEditor flex-1 overflow-auto">
        <div className="text-basis px-4 font-mono text-sm font-light">{children}</div>
      </div>
    </div>
  );
}
