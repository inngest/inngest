import { type CodeBlockAction } from '../CodeBlock';
import { NewCodeBlock } from '../NewCodeBlock/NewCodeBlock';

export type IOProps = {
  title: string;
  actions?: CodeBlockAction[];
  raw?: string;
  error?: boolean;
  loading?: boolean;
};

export const IO = ({ title, actions, raw, error, loading }: IOProps) => {
  return (
    <div className="text-muted bg-codeEditor h-full">
      <NewCodeBlock
        actions={actions}
        header={{ title, ...(error && { status: 'error' }) }}
        tab={{
          content: raw ?? 'Unknown',
        }}
        allowFullScreen={true}
        loading={loading}
      />
    </div>
  );
};
