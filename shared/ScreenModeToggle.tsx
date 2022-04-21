import React, { useState } from "react";
import styled from "@emotion/styled";

import { titleCase } from "./util";

const ScreenModeToggle: React.FC = () => {
  const [mode, setMode] = useState("dark");
  const toggleDarkMode = () => {
    setMode(mode === "dark" ? "light" : "dark");
    document.body.className = document.body.className.includes("dark")
      ? "light-mode"
      : "dark-mode";
  };
  return (
    <ToggleButton mode={mode} onClick={() => toggleDarkMode()}>
      {titleCase(mode)}
    </ToggleButton>
  );
};

const ToggleButton = styled.button<{ mode: string }>`
  padding: 0.1em 0.6em;
  border: 1px solid var(--color-iris-60);
  border-radius: var(--border-radius);
  color: var(--color-iris-60);
  background-color: transparent;

  &:hover {
    background-color: ${({ mode }) =>
      mode === "dark" ? "var(--bg-color-l)" : "var(--bg-color-d)"};
  }
`;

export default ScreenModeToggle;
