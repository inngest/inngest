'use client';

import { forwardRef } from 'react';
import ReactSyntaxHighlighter from 'react-syntax-highlighter';
import { atomOneDark } from 'react-syntax-highlighter/dist/cjs/styles/hljs';

import cn from '@/utils/cn';

type SyntaxHighlighterProps = {
  language: string;
  children: string;
  className?: string;
};

const colors = {
  indigo: '#818cf8',
  green: '#2dd4bf',
  amber: '#ffb74f',
};

const theme = {
  ...atomOneDark,
  'hljs-attr': { color: colors.indigo },
  'hljs-string': { color: colors.green },
  'hljs-number': { color: colors.amber },
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
