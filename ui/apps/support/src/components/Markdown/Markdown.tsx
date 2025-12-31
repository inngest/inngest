import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeRaw from "rehype-raw";
import type { Components } from "react-markdown";
import { remarkParsePlainAttachments } from "./remarkParsePlainAttachments";
import { rehypePreserveHProperties } from "./rehypePreserveHProperties";
import { Attachment } from "@/components/Support/Attachment";
import { SyntaxHighlighter } from "./SyntaxHighlighter";

type MarkdownProps = {
  content: string;
  className?: string;
};

const components: Partial<Components> = {
  // Headings
  h1: ({ children }) => (
    <h1 className="text-basis mb-4 mt-6 text-2xl font-bold">{children}</h1>
  ),
  h2: ({ children }) => (
    <h2 className="text-basis mb-3 mt-5 text-xl font-semibold">{children}</h2>
  ),
  h3: ({ children }) => (
    <h3 className="text-basis mb-2 mt-4 text-lg font-semibold">{children}</h3>
  ),
  h4: ({ children }) => (
    <h4 className="text-basis mb-2 mt-3 text-base font-semibold">{children}</h4>
  ),
  h5: ({ children }) => (
    <h5 className="text-basis mb-1 mt-2 text-sm font-semibold">{children}</h5>
  ),
  h6: ({ children }) => (
    <h6 className="text-muted mb-1 mt-2 text-sm font-semibold">{children}</h6>
  ),

  // Paragraphs
  p: ({ children }) => <p className="text-basis mb-4">{children}</p>,

  // Links
  a: ({ href, children }) => (
    <a
      href={href}
      className="text-link hover:text-linkHover underline decoration-1 underline-offset-2 transition-colors"
      target="_blank"
      rel="noopener noreferrer"
    >
      {children}
    </a>
  ),

  // Lists
  ul: ({ children }) => (
    <ul className="text-basis mb-4 ml-6 list-disc space-y-1">{children}</ul>
  ),
  ol: ({ children }) => (
    <ol className="text-basis mb-4 ml-6 list-decimal space-y-1">{children}</ol>
  ),
  li: ({ children }) => <li className="text-basis">{children}</li>,

  // Code blocks
  code: ({ children, className }) => {
    const isInline = !className;
    // Extract language from className (e.g., "language-javascript" -> "javascript")
    const match = /language-(\w+)/.exec(className || "");
    const language = match ? match[1] : "";

    if (isInline) {
      return (
        <code className="bg-canvasMuted text-contrast rounded px-1.5 py-0.5 font-mono text-sm">
          {children}
        </code>
      );
    }

    // Use syntax highlighter for code blocks with a language
    const codeString = String(children).replace(/\n$/, "");
    if (language) {
      return (
        <SyntaxHighlighter language={language}>{codeString}</SyntaxHighlighter>
      );
    }

    // Fallback for code blocks without a language
    return (
      <code className="text-contrast block font-mono text-sm">{children}</code>
    );
  },
  pre: ({ children }) => (
    <pre className="bg-[rgb(var(--color-carbon-950))] mb-4 overflow-x-auto rounded-lg p-4">
      {children}
    </pre>
  ),

  // Blockquotes
  blockquote: ({ children }) => (
    <blockquote className="border-subtle text-muted my-4 border-l-4 pl-4 italic">
      {children}
    </blockquote>
  ),

  // Tables
  table: ({ children }) => (
    <div className="mb-4 overflow-x-auto">
      <table className="border-subtle w-full border-collapse border">
        {children}
      </table>
    </div>
  ),
  thead: ({ children }) => (
    <thead className="bg-canvasSubtle">{children}</thead>
  ),
  tbody: ({ children }) => <tbody>{children}</tbody>,
  tr: ({ children }) => <tr className="border-subtle border-b">{children}</tr>,
  th: ({ children }) => (
    <th className="border-subtle text-basis border px-4 py-2 text-left font-semibold">
      {children}
    </th>
  ),
  td: ({ children }) => (
    <td className="border-subtle text-basis border px-4 py-2">{children}</td>
  ),

  // Horizontal rule
  hr: () => <hr className="border-subtle my-6 border-t" />,

  // Strong and emphasis
  strong: ({ children }) => (
    <strong className="font-semibold">{children}</strong>
  ),
  em: ({ children }) => <em className="italic">{children}</em>,

  // Strikethrough (from GFM)
  del: ({ children }) => <del className="line-through">{children}</del>,

  img: ({ src, alt, ...props }) => {
    // Plain inline attachments are only attachment ids - they need to be fetched
    const attachmentId = (props as any)["data-attachment-id"];

    if (attachmentId) {
      return <Attachment attachmentId={attachmentId} />;
    }

    return (
      <span className="text-muted text-sm italic">
        Could not display inline image
      </span>
    );
  },
};

export function Markdown({ content, className = "" }: MarkdownProps) {
  return (
    <div className={`prose prose-sm max-w-none ${className}`}>
      <ReactMarkdown
        remarkPlugins={[remarkParsePlainAttachments, remarkGfm]}
        rehypePlugins={[rehypePreserveHProperties, rehypeRaw]}
        components={components}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
