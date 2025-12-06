import { useEffect, useState } from "react";

type KeyCombo = {
  alt?: boolean;
  key: string;
  metaOrCtrl?: boolean;
  shift?: boolean;
};

type KeyboardShortcutTooltipProps = {
  combo: KeyCombo;
};

export function KeyboardShortcutTooltip({
  combo,
}: KeyboardShortcutTooltipProps) {
  const [isMac, setIsMac] = useState(false);

  useEffect(() => {
    const userAgent = navigator.userAgent.toUpperCase();
    setIsMac(userAgent.indexOf("MAC") >= 0);
  }, []);

  const parts: string[] = [];

  if (combo.metaOrCtrl) {
    parts.push(isMac ? "⌘" : "Ctrl");
  }

  if (combo.alt) {
    parts.push(isMac ? "⌥" : "Alt");
  }

  if (combo.shift) {
    parts.push(isMac ? "⇧" : "Shift");
  }

  // Format the key nicely
  const keyName = combo.key.length === 1 ? combo.key.toUpperCase() : combo.key;
  parts.push(keyName);

  return <span>{parts.join("+")}</span>;
}
