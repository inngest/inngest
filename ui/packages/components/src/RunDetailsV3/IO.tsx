import { CodeBlock, type CodeBlockAction } from '../CodeBlock';

export type IOProps = {
  title: string;
  actions?: CodeBlockAction[];
  raw?: string;
  error?: boolean;
};

export const IO = ({ title, actions, raw, error }: IOProps) => {
  return (
    <div className="text-muted h-full overflow-y-scroll" onWheel={(e) => e.stopPropagation()}>
      <CodeBlock
        actions={actions}
        header={{ title, ...(error && { status: 'error' }) }}
        tab={{
          content: raw ?? 'Unknown',
        }}
        alwaysFullHeight={true}
        allowFullScreen={true}
      />
    </div>
  );
};
