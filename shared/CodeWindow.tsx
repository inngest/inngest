import SyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark } from "react-syntax-highlighter/dist/cjs/styles/hljs";

export const removeLeadingSpaces = (snippet: string): string => {
  const lines = snippet.split("\n");
  if (!lines[0]?.replace(/^\s+/, "").length) {
    lines.shift();
  }
  if (!lines[lines.length - 1]?.replace(/^\s+/, "").length) {
    lines.pop();
  }
  const leadingSpace = lines[0]?.match(/^\s+/)?.[0];
  return lines.map((l) => l.replace(leadingSpace, "")).join("\n");
};

const colors = {
  slate300: "rgb(203, 213, 225)",
  fuchsia300: "rgb(240, 171, 252)",
  amber300: "rgb(252, 211, 77)",
  amber400: "rgb(251, 191, 36)",
  sky300: "rgb(125, 211, 252)",
  emerald300: "rgb(110, 231, 183)",
};

const theme = {
  ...atomOneDark,
  "hljs-keyword": { color: colors.fuchsia300 },
  "hljs-attr": { color: colors.amber400 },
  "hljs-string": { color: colors.emerald300 },
  "hljs-number": { color: colors.sky300 },
  "hljs-comment": { color: colors.slate300 },
};

const CodeWindow = ({
  snippet,
  className = "",
  style = {},
  header,
  showLineNumbers = false,
  lineHighlights = [],
}: {
  snippet: string;
  className?: string;
  style?: object;
  header?: React.ReactNode;
  showLineNumbers?: boolean;
  lineHighlights?: [number, number][];
}) => {
  return (
    <div
      className={`rounded-lg border border-slate-700/30 text-xs leading-relaxed bg-slate-800/50 ${className}`}
      style={style}
    >
      {header && <div className="mb-1 bg-slate-800/50">{header}</div>}
      <div className="flex flex-row p-2 items-stretch">
        {Boolean(lineHighlights?.length) && (
          <div className="h-full w-[2px] py-1 relative">
            {/* leading-relaxed is 1.625 */}
            {lineHighlights.map(([highlightStart, highlightEnd]) => {
              return (
                <span
                  className="absolute border-r-2 border-slate-200/50"
                  style={{
                    top: `${(highlightStart - 1) * 1.625 + 0.25 + 0.05}em`, // 0.25 to match top padding + extra nudge
                    height: `${(highlightEnd - highlightStart + 1) * 1.625}em`,
                  }}
                />
              );
            })}
          </div>
        )}
        <SyntaxHighlighter
          language="javascript"
          showLineNumbers={showLineNumbers}
          lineNumberContainerStyle={{
            borderRight: "1px solid pink",
            background: "pink",
          }}
          lineNumberStyle={{
            minWidth: "3em",
            color: "rgba(42, 60, 85, 1)",
          }}
          style={theme}
          customStyle={{
            padding: "0.25rem",
            color: colors.slate300,
            background: "transparent",
          }}
        >
          {removeLeadingSpaces(snippet)}
        </SyntaxHighlighter>
      </div>
    </div>
  );
};

export default CodeWindow;
