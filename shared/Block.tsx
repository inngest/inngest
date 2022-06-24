import React from "react";
import styled from "@emotion/styled";

const COLORS = {
  default: "--black",
  primary: "--primary-color",
};

const Block = styled.div<{
  color?: "default" | "primary";
}>`
  padding: 2rem;
  background: var(
    ${(props) => (props.color ? COLORS[props.color] : COLORS.default)}
  );
  color: var(--color-white);
  border-radius: var(--border-radius);

  h1,
  h2,
  h3,
  h4,
  h5,
  p {
    font-family: var(--font);
  }
`;

export default Block;
