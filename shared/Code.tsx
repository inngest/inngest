import React, { useState } from "react";
import styled from "@emotion/styled";
import { highlight } from "../utils/code";

type Props = {
  code: { [language: string]: string };
  selected?: string;
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
        <code>{props.code[selected]}</code>
      </pre>
    </Wrapper>
  );
};

export default Code;

const Wrapper = styled.div`
  background: var(--black);
  padding: 2rem;
  border-radius: var(--border-radius);

  ul {
    list-style: none;
    display: flex;
    margin: 0 0 2rem;
    padding: 0;
  }

  li button {
    padding: 0.5rem 0.75rem;
    border: 0;
    background: transparent;
    color: #fff;
    border-radius: var(--border-radius);
  }

  li + li { margin: 0 0 0 1rem; }

  li.selected button {
    background: var(--primary-color);
  }

  pre,
  code {
    font-size: 18px;
  }

  @media (max-width: 800px) {
    padding: 1rem;
    li button {
      font-size: .9rem;
      padding: .25rem .5rem;
    }
    li + li { margin: 0 0 0 .5rem; }

    pre,
    code {
      font-size: 14px;
    }
  }
`;
