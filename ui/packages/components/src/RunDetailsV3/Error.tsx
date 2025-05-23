import { CodeBlock, type CodeBlockAction } from '../CodeBlock';

export type ErrorProps = { actions?: CodeBlockAction[]; raw?: string };

export const Error = ({ actions, raw }: ErrorProps) => {
  return (
    <div className="text-muted">
      <CodeBlock
        actions={actions}
        header={{
          title: 'Output',
        }}
        tab={{
          content: raw ?? 'Unknown',
        }}
        allowFullScreen={true}
      />
    </div>
  );
};
