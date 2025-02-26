import { CodeBlock, type CodeBlockAction } from '../CodeBlock';

export type IOProps = { title: string; actions?: CodeBlockAction[]; raw?: string; error?: boolean };

export const IO = ({ title, actions, raw, error }: IOProps) => {
  return (
    <div className="text-muted z-[1]">
      <CodeBlock
        actions={actions}
        header={{ title, ...(error && { status: 'error' }) }}
        tab={{
          content: raw ?? 'Unknown',
        }}
        allowFullScreen={true}
      />
    </div>
  );
};
