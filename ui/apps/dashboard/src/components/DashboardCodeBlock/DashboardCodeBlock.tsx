'use client';

import type { ComponentProps } from 'react';
import { useClerk } from '@clerk/nextjs';
import { CodeBlock } from '@inngest/components/CodeBlock';

/**
 * This component is a wrapper around the CodeBlock component from @inngest/components. It is a
 * workaround for a bug between Monaco and Clerk.
 *
 * @see {@link https://github.com/clerk/javascript/issues/1643} related Clerk issue
 * @see {@link https://clerk.com/docs/troubleshooting/script-loading} related Clerk documentation
 */
export default function DashboardCodeBlock(props: ComponentProps<typeof CodeBlock>) {
  const clerk = useClerk();

  if (!clerk.loaded) return;

  return (
    <CodeBlock.Wrapper>
      <CodeBlock {...props} />
    </CodeBlock.Wrapper>
  );
}
