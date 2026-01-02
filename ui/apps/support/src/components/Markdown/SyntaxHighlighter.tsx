import ReactSyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark } from "react-syntax-highlighter/dist/esm/styles/hljs";
import { cn } from "@inngest/components/utils/classNames";

type SyntaxHighlighterProps = {
  language: string;
  children: string;
  className?: string;
};

// Custom theme based on atomOneDark with some adjustments
const theme = {
  ...atomOneDark,
  hljs: {
    ...atomOneDark.hljs,
    background: "transparent",
  },
};

export function SyntaxHighlighter({
  language,
  children,
  className,
}: SyntaxHighlighterProps) {
  return (
    <ReactSyntaxHighlighter
      language={language}
      showLineNumbers={false}
      style={theme}
      customStyle={{ backgroundColor: "transparent", padding: 0, margin: 0 }}
      className={cn("font-mono text-sm", className)}
    >
      {children}
    </ReactSyntaxHighlighter>
  );
}
