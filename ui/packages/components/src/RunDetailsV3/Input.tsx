import { CodeBlock, type CodeBlockAction } from '../CodeBlock';

export type InputProps = { actions?: CodeBlockAction[]; title?: string; raw?: string };

export const Input = ({ title, actions, raw }: InputProps) => {
  return (
    <div className="text-muted">
      <CodeBlock
        actions={actions}
        header={{
          title: title || 'Input',
        }}
        tab={{
          content: raw ?? 'Unknown',
        }}
        allowFullScreen={true}
      />
    </div>
  );
};
