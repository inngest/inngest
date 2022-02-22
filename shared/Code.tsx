import React, { useState } from "react";
import styled from "@emotion/styled";
import Highlight from "react-highlight.js";

type Props = {
  code: { [language: string]: string };
  selected?: string;
};

const HIJS_LANGUAGES = {
  curl: "bash",
  javascript: "javascript",
  go: "go",
};

const Code: React.FC<Props> = (props) => {
  const langs = Object.keys(props.code);
  const [selected, setSelected] = useState(props.selected || langs[0]);

  return (
    <Wrapper>
      {langs.length > 1 && (
        <ul>
          {langs.map((lang: string) => (
            <li className={lang === selected ? "selected" : ""} key={lang}>
              <button onClick={() => setSelected(lang)}>{lang}</button>
            </li>
          ))}
        </ul>
      )}
      <pre>
        <Highlight language={HIJS_LANGUAGES[selected.toLowerCase()]}>
          {props.code[selected]}
        </Highlight>
      </pre>
    </Wrapper>
  );
};

export default Code;

const Wrapper = styled.div`
  background: var(--black);
  padding: 1.5em;
  border-radius: var(--border-radius);
  font-family: var(--font-mono);

  .hljs {
    background-color: var(--black);
  }
  .hljs-string {
    color: var(--green);
  }

  ul {
    list-style: none;
    display: flex;
    margin: 0 0 1.5rem;
    padding: 0;
    font-size: 0.8rem;
  }

  li button {
    padding: 0.2rem 0.6rem;
    border: 0;
    background: transparent;
    color: #c4c4c4;
    border-radius: var(--border-radius);
    font-weight: bold;

    &:hover {
      background: rgba(var(--primary-color-rgb), 0.3);
    }
  }

  li + li {
    margin: 0 0 0 1rem;
  }

  li.selected button {
    background: var(--primary-color);
    color: #fff;
  }

  pre,
  code {
    font-size: 18px;
  }

  @media (max-width: 800px) {
    padding: 1rem;
    li button {
      font-size: 0.9rem;
      padding: 0.25rem 0.5rem;
    }
    li + li {
      margin: 0 0 0 0.5rem;
    }

    pre,
    code {
      font-size: 14px;
    }
  }
`;
