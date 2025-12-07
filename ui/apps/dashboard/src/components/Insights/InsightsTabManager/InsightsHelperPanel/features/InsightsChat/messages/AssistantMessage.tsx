import type { TextUIPart } from "@inngest/use-agent";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

type AssistantMessageProps = {
  part: TextUIPart;
};

export const AssistantMessage = ({ part }: AssistantMessageProps) => {
  return (
    <div className="text-basis prose prose-sm max-w-full rounded-md px-0 py-1">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          // Customize paragraph styling
          p: ({ children }) => (
            <p className="my-2 text-sm leading-relaxed">{children}</p>
          ),
          // Customize list styling
          ul: ({ children }) => (
            <ul className="my-2 ml-4 list-disc text-sm">{children}</ul>
          ),
          ol: ({ children }) => (
            <ol className="my-2 ml-4 list-decimal text-sm">{children}</ol>
          ),
          li: ({ children }) => <li className="my-1">{children}</li>,
          // Customize code styling
          code: ({ children, className, ...props }) => {
            const isInline = !className || !className.includes("language-");
            if (isInline) {
              return (
                <code
                  className="bg-canvasSubtle text-muted rounded px-1 py-0.5 font-mono text-xs"
                  {...props}
                >
                  {children}
                </code>
              );
            }
            return (
              <code
                className="bg-canvasSubtle text-muted block overflow-x-auto rounded p-2 font-mono text-xs"
                {...props}
              >
                {children}
              </code>
            );
          },
          // Customize pre (code block wrapper) styling
          pre: ({ children }) => (
            <pre className="bg-canvasSubtle my-2 overflow-x-auto rounded p-2">
              {children}
            </pre>
          ),
          // Customize heading styling
          h1: ({ children }) => (
            <h1 className="mb-2 mt-3 text-base font-bold">{children}</h1>
          ),
          h2: ({ children }) => (
            <h2 className="mb-2 mt-3 text-sm font-bold">{children}</h2>
          ),
          h3: ({ children }) => (
            <h3 className="mb-2 mt-2 text-sm font-semibold">{children}</h3>
          ),
          // Customize link styling
          a: ({ children, href }) => (
            <a
              href={href}
              className="text-link underline hover:opacity-80"
              target="_blank"
              rel="noopener noreferrer"
            >
              {children}
            </a>
          ),
          // Customize blockquote styling
          blockquote: ({ children }) => (
            <blockquote className="border-subtle text-muted my-2 border-l-4 pl-4 italic">
              {children}
            </blockquote>
          ),
          // Customize table styling
          table: ({ children }) => (
            <table className="my-2 w-full border-collapse text-sm">
              {children}
            </table>
          ),
          thead: ({ children }) => (
            <thead className="bg-canvasSubtle">{children}</thead>
          ),
          tbody: ({ children }) => <tbody>{children}</tbody>,
          tr: ({ children }) => (
            <tr className="border-subtle border-b">{children}</tr>
          ),
          th: ({ children }) => (
            <th className="border-subtle border px-2 py-1 text-left font-semibold">
              {children}
            </th>
          ),
          td: ({ children }) => (
            <td className="border-subtle border px-2 py-1">{children}</td>
          ),
          // Customize horizontal rule styling
          hr: () => <hr className="border-subtle my-3 border-t" />,
          // Customize strong/bold styling
          strong: ({ children }) => (
            <strong className="font-semibold">{children}</strong>
          ),
          // Customize emphasis/italic styling
          em: ({ children }) => <em className="italic">{children}</em>,
        }}
      >
        {part.content}
      </ReactMarkdown>
    </div>
  );
};
