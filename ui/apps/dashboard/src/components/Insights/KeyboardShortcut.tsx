"use client";

import { useEffect, useState } from "react";
import { cn } from "@inngest/components/utils/classNames";
import { RiCommandLine } from "@remixicon/react";

type KeyboardShortcutProps = {
  backgroundColor?: string;
  className?: string;
  color?: string;
  keys: Array<"cmd" | "ctrl" | "alt" | "shift" | "enter" | string>;
};

export function KeyboardShortcut({
  backgroundColor,
  className,
  color,
  keys,
}: KeyboardShortcutProps) {
  const [isMac, setIsMac] = useState(false);

  useEffect(() => {
    const userAgent = navigator.userAgent.toUpperCase();
    setIsMac(userAgent.indexOf("MAC") >= 0);
  }, []);

  const renderKey = (key: string, index: number) => {
    const normalizedKey = key.toLowerCase();

    // Handle platform-specific modifier keys
    if (normalizedKey === "cmd" || normalizedKey === "ctrl") {
      if (isMac && normalizedKey === "cmd") {
        return <RiCommandLine key={index} className="h-4 w-4" />;
      }
      if (!isMac && normalizedKey === "ctrl") {
        return (
          <span key={index} className="text-xs font-semibold">
            Ctrl
          </span>
        );
      }
      // If key doesn't match platform, skip it
      return null;
    }

    // Handle other modifier keys
    if (normalizedKey === "alt") {
      return (
        <span key={index} className="text-xs font-semibold">
          {isMac ? "⌥" : "Alt"}
        </span>
      );
    }

    if (normalizedKey === "shift") {
      return (
        <span key={index} className="text-xs font-semibold">
          {isMac ? "⇧" : "Shift"}
        </span>
      );
    }

    // Handle special keys with icons or symbols
    if (normalizedKey === "enter") {
      return (
        <span key={index} className="text-xs font-semibold">
          {isMac ? "⏎" : "↵"}
        </span>
      );
    }

    // Default: render as uppercase text
    return (
      <span key={index} className="text-xs font-semibold">
        {key.toUpperCase()}
      </span>
    );
  };

  const renderedKeys = keys.map(renderKey).filter(Boolean);

  // Determine background color - default to transparent if not provided
  const bgColor =
    backgroundColor !== undefined ? backgroundColor : "bg-transparent";

  return (
    <div
      className={cn(
        "flex shrink-0 items-center gap-0.5 rounded-[4px] px-1 py-0.5",
        bgColor,
        color,
        className,
      )}
    >
      {renderedKeys.map((key, index) => (
        <span key={index}>{key}</span>
      ))}
    </div>
  );
}
