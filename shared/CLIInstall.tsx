import React, { useState } from "react";
import styled from "@emotion/styled";

const SCRIPT = "curl -sfL https://cli.inngest.com/install.sh | sh";

const CLIInstall: React.FC<{
  theme?: "gradient";
}> = ({ theme = "gradient" }) => {
  const [buttonText, setButtonText] = useState("Copy");
  const handleClickCopy = async () => {
    await navigator?.clipboard?.writeText(SCRIPT);
    setButtonText("Copied!");
  };
  return (
    <Code className="cli-install">
      <pre>{SCRIPT}</pre>
      <CopyButton onClick={handleClickCopy}>{buttonText}</CopyButton>
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
  border-radius: var(--border-radius);
  background-color: rgba(7, 7, 7, 0.1);
  border: 0;
  white-space: nowrap;
`;

export default CLIInstall;
