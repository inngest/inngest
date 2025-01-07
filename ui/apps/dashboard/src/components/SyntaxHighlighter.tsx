'use client';

import { forwardRef } from 'react';
import { cn } from '@inngest/components/utils/classNames';
import ReactSyntaxHighlighter from 'react-syntax-highlighter';
import { atomOneDark } from 'react-syntax-highlighter/dist/cjs/styles/hljs';
import colors from 'tailwindcss/colors';

type SyntaxHighlighterProps = {
  language: string;
  children: string;
  className?: string;
};

const theme = {
  ...atomOneDark,
  'hljs-attr': { color: colors.indigo['400'] },
  'hljs-string': { color: colors.teal['400'] },
  'hljs-number': { color: colors.amber['300'] },
};

const SyntaxHighlighter = forwardRef(function SyntaxHighlighter(
  { language, children, className }: SyntaxHighlighterProps,
  ref: React.ForwardedRef<HTMLPreElement>
) {
  const PreWithRef = (preProps: React.ComponentProps<'pre'>) => <pre {...preProps} ref={ref} />;

  return (
    <ReactSyntaxHighlighter
      PreTag={PreWithRef}
      language={language}
      showLineNumbers={false}
      style={theme}
      customStyle={{ backgroundColor: 'transparent' }}
      className={cn('font-mono text-sm font-light', className)}
    >
      {children}
    </ReactSyntaxHighlighter>
  );
});

export default SyntaxHighlighter;
