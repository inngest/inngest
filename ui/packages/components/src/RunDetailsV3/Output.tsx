import { CodeBlock, type CodeBlockAction } from '../CodeBlock';

export type OutputProps = { actions?: CodeBlockAction[]; raw?: string };

export const Output = ({ actions, raw }: OutputProps) => {
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
