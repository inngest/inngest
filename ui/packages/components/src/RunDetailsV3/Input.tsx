import { CodeBlock, type CodeBlockAction } from '../CodeBlock';

export type InputProps = { actions?: CodeBlockAction[]; raw?: string };

export const Input = ({ actions, raw }: InputProps) => {
  return (
    <div className="text-muted">
      <CodeBlock
        actions={actions}
        header={{
          title: 'Function Payload',
        }}
        tab={{
          content: raw ?? 'Unknown',
        }}
        allowFullScreen={true}
      />
    </div>
  );
};
