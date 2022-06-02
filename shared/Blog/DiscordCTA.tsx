import React from "react";
import styled from "@emotion/styled";

import Button from "../Button";
import DiscordLogo from "../Icons/Discord";

const DiscordCTA: React.FC = () => {
  return (
    <Box>
      <DiscordLogo />
      <p>Ask questions, give feedback, and share feature requests</p>
      <Button
        href={process.env.NEXT_PUBLIC_DISCORD_URL}
        kind="outlineHighContrast"
      >
        Join our Discord!
      </Button>
    </Box>
  );
};
const Box = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 2rem 0;
  background-color: #5865f2; // Discord Blurple
  color: #fff;
  border-radius: var(--border-radius);
  text-align: center;

  svg {
    font-size: 120px;
  }
  p {
    margin: 1.5rem 0;
  }
`;
export default DiscordCTA;
