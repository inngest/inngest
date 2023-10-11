type TransformEditorProps = {
  type: 'incoming' | 'transformed';
  children: React.ReactNode;
};

export function TransformEditor({ type, children }: TransformEditorProps) {
  let label = '';
  let title = '';
  switch (type) {
    case 'incoming':
      title = 'Incoming Event JSON';
      label = 'Paste the incoming JSON payload here to test your transform.';
      break;
    case 'transformed':
      title = 'Transformed Event';
      label = 'Preview the transformed JSON payload here.';
      break;
  }

  return (
    <div className="bg-slate-950 flex h-full w-6/12 flex-col rounded-lg text-white">
      <header className="rounded-t-lg border-b border-slate-800 bg-slate-900 px-5 py-3">
        <h2 className="text-base font-medium">{title}</h2>
        <p className="mt-0.5 text-sm font-light tracking-wide text-slate-300">{label}</p>
      </header>
      <div className="flex-1 overflow-auto">
        <div className="px-6 py-2 font-mono text-sm font-light text-white">{children}</div>
      </div>
    </div>
  );
}
