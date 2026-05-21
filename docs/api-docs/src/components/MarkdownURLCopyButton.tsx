"use client";
import { type ComponentProps, useState } from "react";
import { Check, Copy } from "lucide-react";
// import { cn } from "fumadocs-ui/utils/cn";
import { useCopyButton } from "fumadocs-ui/utils/use-copy-button";
import { buttonVariants } from "fumadocs-ui/components/ui/button";

const cache = new Map<string, Promise<string>>();

export function MarkdownURLCopyButton({
  markdownUrl,
  ...props
}: ComponentProps<"button"> & {
  /**
   * A URL to fetch the raw Markdown/MDX content of page
   */
  markdownUrl: string;
}) {
  const [isLoading, setLoading] = useState(false);
  const [checked, onClick] = useCopyButton(async () => {
    const cached = cache.get(markdownUrl);
    if (cached) return navigator.clipboard.writeText(await cached);

    setLoading(true);

    try {
      const promise = fetch(markdownUrl).then((res) => res.text());
      cache.set(markdownUrl, promise);
      await navigator.clipboard.write([
        new ClipboardItem({
          "text/plain": promise,
        }),
      ]);
    } finally {
      setLoading(false);
    }
  });

  return (
    <button
      disabled={isLoading}
      onClick={onClick}
      {...props}
      className={buttonVariants({
        color: "secondary",
        size: "sm",
        className: "gap-2 [&_svg]:size-3.5 [&_svg]:text-fd-muted-foreground",
      })}
    >
      {checked ? <Check /> : <Copy />}
      {props.children ?? "Copy Markdown URL"}
    </button>
  );
}
