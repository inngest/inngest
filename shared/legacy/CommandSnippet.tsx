import styled from "@emotion/styled";
import { useCallback, useState } from "react";

interface Props {
  command: string;
  copy?: boolean;
}

export const CommandSnippet: React.FC<Props> = ({ command, copy }) => {
  const [buttonText, setButtonText] = useState("Copy");
  const handleClickCopy = useCallback(async () => {
    await navigator?.clipboard?.writeText(command);
    setButtonText("Copied!");
  }, [command]);
  return (
    <Code className="cli-install">
      <pre>{command}</pre>

      {copy ? (
        <CopyButton onClick={handleClickCopy}>{buttonText}</CopyButton>
      ) : null}
    </Code>
  );
};

const Code = styled.code`
  display: inline-flex;
  align-items: center;
  padding: 0.6em 1em;
  font-size: 0.7rem;
  background: linear-gradient(
    135deg,
    hsl(332deg 30% 95%) 0%,
    hsl(240deg 30% 95%) 100%
  );
  color: var(--black);
  border-radius: 10px;
  border: 2px solid rgba(7, 7, 7, 0.1);
  font-weight: bold;
`;

const CopyButton = styled.button`
  margin-left: 1em;
  padding: 0 0.2rem;
  border-radius: var(--border-radius);
  background-color: rgba(7, 7, 7, 0.1);
  border: 0;
  white-space: nowrap;
`;
