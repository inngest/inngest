// @ts-nocheck
import React, { useState, useEffect } from "react";
import styled from "@emotion/styled";
import { css } from "@emotion/react";

type Props = {
  onClose: () => void;
  footer?: React.ReactNode;
  children: React.ReactNode;
};

const Modal: React.FC<Props> = ({ onClose, children, footer }) => {
  const [faded, setFaded] = useState(true);
  useEffect(() => setFaded(false), []);

  // TODO: Listen for esc key press

  // Show the faded class then hide.
  const onFadeClose = () => {
    setFaded(true);
    window.setTimeout(onClose, 200);
  };

  const preventClose = (e: React.SyntheticEvent) => {
    e.preventDefault();
    e.stopPropagation();
  };

  return (
    <BG onClick={onFadeClose} css={[faded && fadedCss]}>
      <div onClick={preventClose}>{children}</div>
      {footer || null}
    </BG>
  );
};

const fadedCss = css`
  opacity: 0;
  transform: translateY(-10px);
  > div {
    transform: scale(0.95);
  }
`;

const BG = styled.div`
  z-index: 999;
  position: fixed;
  height: 100vh;
  width: 100vw;
  top: 0;
  left: 0;
  background: rgba(0, 0, 0, 0.15);
  opacity: 1;
  transition: all 0.2s;

  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;

  > div:first-of-type {
    transition: all 0.2s;
    box-shadow: 0 8px 15px rgba(0, 0, 0, 0.12);
  }

  > .buttongroup {
    margin-top: 20px;
    flex: 0;
  }
`;

export default Modal;
