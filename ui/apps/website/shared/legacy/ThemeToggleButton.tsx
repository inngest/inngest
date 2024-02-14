import React, { useEffect, useState } from "react";
import styled from "@emotion/styled";

import { titleCase } from "./util";

const ThemeToggleButton: React.FC<{ isFloating?: boolean }> = ({
  isFloating = false,
}) => {
  const [isOpen, setOpen] = useState(false);
  const [theme, setTheme] = useState("dark");

  const getOppositeTheme = (theme: string): "light" | "dark" =>
    theme === "light" ? "dark" : "light";

  const updateThemeAndSave = (newTheme: "light" | "dark") => {
    document.body.classList.remove(`${getOppositeTheme(newTheme)}-theme`);
    document.body.classList.add(`${newTheme}-theme`);
    setTheme(newTheme);
    window.localStorage.setItem("theme", newTheme);
  };
  useEffect(() => {
    const savedTheme = window.localStorage.getItem("theme");
    if (savedTheme) {
      setTheme(savedTheme);
    }
  });
  const options = theme === "light" ? ["light", "dark"] : ["dark", "light"];
  return (
    <ToggleButton
      className="theme-button"
      theme={theme}
      isOpen={isOpen}
      isFloating={isFloating}
      onClick={() => setOpen(!isOpen)}
    >
      {titleCase(theme)}
      <span className="theme-options">
        {options.map((option) => (
          <span
            key={option}
            className={`theme-option ${option}`}
            onClick={() => updateThemeAndSave(option as "light" | "dark")}
          >
            {titleCase(option)}
          </span>
        ))}
      </span>
    </ToggleButton>
  );
};

const ToggleButton = styled.button<{
  theme: string;
  isOpen: boolean;
  isFloating: boolean;
}>`
  --padding: 0.1em 0.2em;

  position: ${({ isFloating }) => (isFloating ? "fixed" : "relative")};
  bottom: ${({ isFloating }) => (isFloating ? "0.5rem" : "auto")};
  right: ${({ isFloating }) => (isFloating ? "0.5rem" : "auto")};
  z-index: 10;
  padding: var(--padding);
  min-width: 58px; // prevent wrapping of "Light" on mobile
  border: var(--button-border-width) solid var(--color-iris-60);
  border-radius: 6px;
  font-size: 14px;
  color: var(--color-iris-60);
  background-color: var(--bg-color);

  /* &:hover {
    background-color: ${({ theme }) =>
    theme === "dark" ? "var(--bg-color-l)" : "var(--bg-color-d)"};
  } */

  .theme-options {
    display: ${({ isOpen }) => (isOpen ? "block" : "none")};
    position: absolute;
    z-index: 20;
    top: ${({ isFloating }) =>
      isFloating ? "auto" : "calc(-1 * var(--button-border-width))"};
    bottom: ${({ isFloating }) =>
      isFloating ? "calc(-1 * var(--button-border-width))" : "auto"};
    left: calc(-1 * var(--button-border-width));
    right: calc(-1 * var(--button-border-width));
    width: calc(100% + 2 * var(--button-border-width));
    overflow: hidden;
    background: var(--bg-color);
    border: var(--button-border-width) solid var(--color-iris-60);
    border-radius: 6px;
  }
  .theme-option {
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

export default ThemeToggleButton;
