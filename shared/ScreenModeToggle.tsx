import React, { useEffect, useState } from "react";
import styled from "@emotion/styled";

import { titleCase } from "./util";

const ScreenModeToggle: React.FC = () => {
  const [isOpen, setOpen] = useState(false);
  const [mode, setMode] = useState("dark");

  const getOppositeMode = (mode: string): "light" | "dark" =>
    mode === "light" ? "dark" : "light";
  const getCurrentMode = () =>
    document.body.className.includes("dark") ? "dark" : "light";

  const updateModeAndSave = (newMode: "light" | "dark") => {
    // const newMode = getOppositeMode(getCurrentMode());
    document.body.className = `${newMode}-mode`;
    setMode(newMode);
    console.log("Update to ", newMode);
    window.localStorage.setItem("screen-mode", newMode);
  };
  useEffect(() => {
    const savedMode = window.localStorage.getItem("screen-mode");
    if (savedMode) {
      setMode(savedMode);
    }
  });
  const options = mode === "light" ? ["light", "dark"] : ["dark", "light"];
  return (
    <ToggleButton mode={mode} isOpen={isOpen} onClick={() => setOpen(!isOpen)}>
      {titleCase(mode)}
      <span className="screen-mode-options">
        {options.map((option) => (
          <span
            key={option}
            className={`screen-mode-option ${option}`}
            onClick={() => updateModeAndSave(option as "light" | "dark")}
          >
            {titleCase(option)}
          </span>
        ))}
      </span>
    </ToggleButton>
  );
};

const ToggleButton = styled.button<{ mode: string; isOpen: boolean }>`
  --padding: 0.1em 0.2em;

  position: relative;
  padding: var(--padding);
  min-width: 58px; // prevent wrapping of "Light" on mobile
  border: var(--button-border-width) solid var(--color-iris-60);
  border-radius: 6px;
  color: var(--color-iris-60);
  background-color: transparent;

  /* &:hover {
    background-color: ${({ mode }) =>
    mode === "dark" ? "var(--bg-color-l)" : "var(--bg-color-d)"};
  } */

  .screen-mode-options {
    display: ${({ isOpen }) => (isOpen ? "block" : "none")};
    position: absolute;
    top: calc(-1 * var(--button-border-width));
    left: calc(-1 * var(--button-border-width));
    right: calc(-1 * var(--button-border-width));
    width: calc(100% + 2 * var(--button-border-width));
    overflow: hidden;
    background: var(--bg-color);
    border: var(--button-border-width) solid var(--color-iris-60);
    border-radius: 6px;
  }
  .screen-mode-option {
    display: block;
    padding: var(--padding);
    border-radius: 4px;

    &.dark {
      background: var(--bg-color-d);
    }
    &.light {
      background: var(--bg-color-l);
    }
  }

  @media (max-width: 800px) {
    min-width: 64px;
    font-size: 16px;
  }
`;

export default ScreenModeToggle;
