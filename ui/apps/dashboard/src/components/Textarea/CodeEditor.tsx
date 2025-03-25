import { useId, useRef, useState } from 'react';
import { cn } from '@inngest/components/utils/classNames';

import SyntaxHighlighter from '@/components/SyntaxHighlighter';

type CodeEditorProps = {
  language: string;
  initialCode?: string;
  onCodeChange?: (code: string) => void;
  label?: string;
  name?: string;
  readOnly?: boolean;
  className?: string;
};

export default function CodeEditor({
  language,
  initialCode = '',
  onCodeChange,
  label,
  name,
  readOnly,
  className,
}: CodeEditorProps) {
  const [code, setCode] = useState(initialCode);
  function handleCodeChange(event: React.ChangeEvent<HTMLTextAreaElement>) {
    const value = event.target.value;
    setCode(value);
    onCodeChange?.(value);
  }

  const codeRef = useRef<HTMLPreElement>(null);
  const LineNumbersRef = useRef<HTMLDivElement>(null);
  function synchronizeScrollPosition(event: React.UIEvent<HTMLTextAreaElement>) {
    if (!codeRef.current || !LineNumbersRef.current) return;
    codeRef.current.scrollTop = event.currentTarget.scrollTop;
    codeRef.current.scrollLeft = event.currentTarget.scrollLeft;
    LineNumbersRef.current.scrollTop = event.currentTarget.scrollTop;
  }

  const textAreaID = useId();
  const numberOfLines = (readOnly ? initialCode : code).split('\n').length;
  const numberOfLinesArray = Array.from({ length: numberOfLines }, (_, i) => i + 1);

  return (
    <div className={cn('flex min-h-[200px] font-mono', className)}>
      <label className="hidden" htmlFor={textAreaID}>
        {label ?? name}
      </label>
      <div
        className="text-subtle overflow-y-hidden py-2 pr-3 text-right text-sm"
        aria-hidden="true"
        ref={LineNumbersRef}
      >
        {numberOfLinesArray.map((lineNumber) => (
          <div key={lineNumber}>{lineNumber}</div>
        ))}
      </div>
      <div className="relative w-full">
        <SyntaxHighlighter
          ref={codeRef}
          aria-hidden={!readOnly}
          language={language}
          className="absolute inset-0 !overflow-auto !py-2 !pl-2 text-sm"
        >
          {readOnly ? initialCode : code}
        </SyntaxHighlighter>
        {!readOnly && (
          <textarea
            className="text-basis bg-codeEditor caret-basis absolute inset-0 z-10 h-full w-full resize-none overflow-auto whitespace-pre border-none py-2 pl-2 font-[CircularXXMono] text-sm leading-5 focus:ring-0 focus-visible:outline-none"
            id={textAreaID}
            name={name}
            value={code}
            onChange={handleCodeChange}
            onScroll={synchronizeScrollPosition}
            spellCheck="false"
          />
        )}
      </div>
    </div>
  );
}
