import { css } from "@emotion/react";
import { nodeW, nodeH } from "../consts";

export const baseCSS = css`
  border-radius: 5px;
  background: #fff;
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.05);
  height: ${nodeH}px;
  width: ${nodeW}px;
  padding: 0 20px;
  border: 1px solid #e8e8e6;
  position: relative;

  transition: all 0.2s;

  p {
    margin: 0;
    font-weight: 500;
  }

  > span {
    font-size: 12px;
  }
`;
