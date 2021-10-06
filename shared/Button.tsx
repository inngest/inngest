import React from "react";
import styled from "@emotion/styled";
import { css, SerializedStyles } from "@emotion/react";

// Button props
type Props = React.HTMLAttributes<any> & {
  kind?: KindStrings;
  size?: SizeStrings;
  loading?: boolean;
  type?: "button" | "reset" | "submit" | undefined;
  onClick?: (() => void) | ((e: React.SyntheticEvent) => any);
  disabled?: boolean;
  link?: string;
  target?: string;
  style?: object;
  className?: string;
  children: React.ReactNode;
};

type ButtonGroupProps = {
  children: React.ReactNode;
  center?: boolean;
  right?: boolean;
  small?: boolean;
  stretch?: boolean;
  style?: any;
  packed?: boolean;
};

export const ButtonGroup = ({
  children,
  right,
  center,
  small,
  stretch,
  style,
  packed,
}: ButtonGroupProps) => (
  <Group
    style={style}
    className="buttongroup"
    css={[
      packed &&
        css`
          > button + button {
            margin-left: -1px;
          }
        `,
      right &&
        css`
          justify-content: flex-end;
        `,
      center &&
        css`
          justify-content: center;
        `,
      stretch &&
        css`
          justify-content: stretch;
          button {
            justify-content: center;
          }
          button:only-child {
            flex: 1;
            width: 100%;
          }
        `,
      small &&
        css`
          font-size: 12.8px;
        `,
    ]}
  >
    {children}
  </Group>
);

export default React.forwardRef<HTMLButtonElement, Props>(
  (props: Props, ref) => {
    const { kind, size, link, children, loading, ...rest } = props;
    let { onClick } = props;

    let C: any = Button;
    // lets us smartly apply "href" to link components
    let cProps = {};

    if (link) {
      C = Link;
      cProps = { href: link };
      onClick = (e: React.SyntheticEvent) => {
        if (props.target !== undefined) {
          // use a normal handler to open a tab if there's target="_blank" etc.
          return;
        }
        if (link.indexOf("://") !== -1) {
          window.location.href = link;
          return;
        }
        e.preventDefault();
      };
    }

    return (
      <C
        {...cProps}
        ref={ref}
        css={[
          kind && kinds[kind],
          size && sizes[size],
          props.disabled && disabled,
          loading &&
            css`
              opacity: 0.75;
            `,
        ]}
        {...rest}
        onClick={onClick}
        className={`button ${props.className || ""}`}
      >
        {loading ? "Loading..." : children}
      </C>
    );
  }
);

export const buttonCSS = css`
  padding: 8px 16px 7px;
  background: #fff;
  border-radius: 3px;
  border: 0 none;
  color: #fff;
  border: 1px solid #ddd;
  box-shadow: 0;
  color: #232222;
  font-size: inherit;
  display: flex;
  text-decoration: none;
  transition: all 0.3s;
  cursor: pointer;
  align-items: center;

  &:hover {
    color: #232222;
    opacity: 1;
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  }

  & + button {
    margin-left: 20px;
  }

  svg,
  img {
    margin: 0 8px 0 -5px;
    align-self: center;
  }
`;

const Link = styled.a`
  ${buttonCSS};
`;

const Button = styled.button`
  ${buttonCSS}
`;

export const primaryCSS = css`
  background: #ceecce;
  background: #bbe6bbee;
  border: 1px solid #93efc3;
  color: #355134;

  &:hover {
    color: #355134;
  }

  svg {
    fill: #fff !important;
  }

  &:disabled {
    opacity: 0.4;
    cursor: disabled;
  }
`;

const defaultCSS = css``;

const dashed = css`
  border-style: dashed;
  border-color: #65c2aa;
  background: transparent;
`;

const link = css`
  border: 1px solid transparent;
  background: transparent;
  box-shadow: none;
  padding: 0;
  color: #0d81cb;

  &:hover {
    border: 1px solid transparent;
    background: transparent;
    color: #0d81cb;
    box-shadow: none;
    transform: none;
  }
`;

const disabled = css`
  opacity: 0.7;
  cursor: not-allowed;
  color: #777;
  border-color: #aaa;

  &:hover {
    box-shadow: none;
    color: #777;
    background: transparent;
    border-color: #aaa;
    transform: none;
  }
`;

const danger = css`
  color: #cb2525;
  border: 1px solid #f78484;

  &:hover {
    border: 1px solid #cb2525;
    background: #cb2525;
    color: #fff;
  }
`;

const outlineWhite = css`
  background: transparent;
  border: 1px solid #ffffff99;
  color: #fff;

  &:hover {
    border: 1px solid #fff;
    background: #ffffff11;
    color: #fff;
  }
`;

const large = css`
  padding: 16px 16px 14px;
  font-size: 1.125rem;
`;

const medium = css`
  padding: 11px 24px 11px;
`;

const small = css`
  padding: 5px 12px 4px;
  font-size: 0.8rem;

  & + button {
    margin-left: 8px;
  }
`;

const Group = styled.div`
  display: flex;
  flex-direction: row;
  align-items: center;
  flex: 1;

  > div + div {
    margin-left: 12px;
  }

  span {
    margin: 0 1rem;
  }

  label + & {
    margin-top: 30px;
  }

  button {
    display: flex;
    align-items: center;
  }
`;

enum Sizes {
  large,
  medium,
  small,
}

type SizeStrings = keyof typeof Sizes;

const sizes: { [key in SizeStrings]: SerializedStyles } = {
  large: large,
  medium: medium,
  small: small,
};

enum Kinds {
  submit,
  primary,
  dashed,
  danger,
  outlineWhite,
  link,
  default,
}

export type KindStrings = keyof typeof Kinds;

const kinds: { [key in KindStrings]: SerializedStyles } = {
  submit: primaryCSS,
  primary: primaryCSS,
  dashed: dashed,
  danger: danger,
  outlineWhite: outlineWhite,
  link: link,
  default: defaultCSS,
};
