// @ts-nocheck
import React from "react";
import styled from "@emotion/styled";
import Button, { ButtonGroup } from "src/shared/Button";
import { css, SerializedStyles } from "@emotion/react";

type Props = {
  kind?: KindStrings;
  style?: any;
  className?: any;
  nopadding?: boolean;
  children: React.ReactNode;
  title?: string | React.ReactNode;

  action?: string;
  onAction?: () => void;
  onClick?: () => void;
};

enum Kinds {
  primary,
  subtleFocus,
  plain,
  dashed,
  hoverable,
  blank,
}

const padding = css`
  padding: 24px;
`;

const noPadding = css`
  padding: 0;
`;

export const BoxWrapper = styled.div`
  ${padding}
  background: #fff;
  border-radius: 5px;
  position: relative;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.05);
  border: 1px solid #e8e8e6;

  canvas {
    pointer-events: none;
    position: absolute;
    z-index: 0;
    height: 100%;
    width: 100%;
    top: 0;
    left: 0;
  }
`;

export const Padding = styled.div`
  ${padding}
`;

const primary = css`
  position: relative;
  padding: 40px 36px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);

  &:before,
  &:after {
    position: absolute;
    display: block;
    content: "";
    z-index: 0;
    top: 0;
    left: 0;
    height: 100%;
    width: 100%;
  }

  &:before {
    background: linear-gradient(0deg, #24a580, #31b18c);
    border-radius: 3px;
    opacity: 0.75;
  }

  &:after {
    margin: 0 0 0 8px;
    background: #fff;
    border-radius: 2px;
  }

  > * {
    position: relative;
    z-index: 1;
  }
`;

const subtleFocus = css`
  box-shadow: 0 3px 10px rgba(0, 0, 0, 0.05);
`;

const dashed = css`
  background: transparent;
  box-shadow: none;
  border: 1px dashed rgba(0, 0, 0, 0.2);
  border-radius: 3px;
  opacity: 0.7;
`;

const plain = css`
  background: #fff;
  box-shadow: none;
  border: 1px solid rgba(0, 0, 0, 0.05);
  border-radius: 3px;
`;

const hoverable = css`
  border: 1px solid transparent;
  box-shadow: none;
  transition: all 0.3s;

  &:hover {
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.03), 0 3px 25px rgba(0, 0, 0, 0.04);
    cursor: pointer;
  }
`;

const blank = css`
  border: 0 none;
  border-radius: 0;
  padding: 30px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #fdfbf6;
`;

const Title = styled.div`
  margin-top: -10px;
  min-height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 0 20px;
  opacity: 0.8;
  font-weight: 600;

  button {
    font-size: 14px;
  }
`;

type KindStrings = keyof typeof Kinds;

const kinds: { [key in KindStrings]: SerializedStyles } = {
  primary: primary,
  dashed: dashed,
  subtleFocus: subtleFocus,
  plain: plain,
  hoverable: hoverable,
  blank: blank,
};

export const Box = (props: Props) => {
  const hasTitle = !!props.title || (!!props.action && !!props.onAction);

  return (
    <BoxWrapper
      css={[props.nopadding && noPadding, props.kind && kinds[props.kind]]}
      className={`${props.className || ""} box`}
      style={props.style}
      onClick={props.onClick}
    >
      {hasTitle && (
        <Title>
          <p>{props.title}</p>
          {props.action && props.onAction && (
            <ButtonGroup right>
              <Button kind="link" onClick={props.onAction}>
                {props.action}
              </Button>
            </ButtonGroup>
          )}
        </Title>
      )}
      {props.children}
    </BoxWrapper>
  );
};

export default Box;
