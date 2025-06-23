import { useRef } from 'react';

import { NewCodeBlock, type CodeBlockAction } from '../NewCodeBlock/NewCodeBlock';

export type IOProps = {
  title: string;
  actions?: CodeBlockAction[];
  raw?: string;
  error?: boolean;
};

export const NewIO = ({ title, actions, raw, error }: IOProps) => {
  const parentRef = useRef<HTMLDivElement>(null);
  return (
    <div ref={parentRef} className="text-muted">
      <NewCodeBlock
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
