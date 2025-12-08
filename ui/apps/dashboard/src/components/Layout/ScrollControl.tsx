"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";

type ScrollControlProps = {
  containerId: string;
};

// TODO: Multiple overflow scrollbars in the same page lead to janky behavior.
// Potential solution:
// Use a single overflow scrollbar for the page and hide it when not needed.
// This would be added to ./Layout.tsx if we decide to go this route.
export default function ScrollControl({ containerId }: ScrollControlProps) {
  const pathname = usePathname();

  // (dylan): Can we do a conditional class application using cn util instead of useEffect?
  useEffect(() => {
    const el = document.getElementById(containerId);
    if (!el) return;

    el.style.overflowY =
      pathname === "/env/production/insights" ? "hidden" : "scroll";

    return () => {
      el.style.overflowY = "";
    };
  }, [containerId, pathname]);

  return null;
}
