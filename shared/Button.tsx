import React from "react";
import styled from "@emotion/styled";
import { css, SerializedStyles } from "@emotion/react";

const Kinds = {
  PRIMARY: "primary",
  OUTLINE: "outline",
  OUTLINE_HIGH_CONTRAST: "outlineHighContrast",
  OUTLINE_PRIMARY: "outlinePrimary",
  BLACK: "black",
};
export type Kinds = typeof Kinds[keyof typeof Kinds];

// Button props
type Props = React.HTMLAttributes<any> & {
  kind?: Kinds;
  type?: "button" | "reset" | "submit" | undefined;
  onClick?: (() => void) | ((e: React.SyntheticEvent) => any);
  disabled?: boolean;
  href?: string;
  target?: string;
  style?: object;
  className?: string;
  size?: "small" | "medium" | "default";
  children: React.ReactNode;
};

export default React.forwardRef<HTMLButtonElement, Props>(
  (props: Props, ref) => {
    const { kind, href, children, ...rest } = props;
    let { onClick } = props;

    let C: any = Button;
    // lets us smartly apply "href" to link components
    let cProps = {};

    if (href) {
      C = Link;
      cProps = { href: href };
      onClick = (e: React.SyntheticEvent) => {
        if (props.target !== undefined) {
          // use a normal handler to open a tab if there's target="_blank" etc.
          return;
        }
        if (href.indexOf("://") !== -1) {
          window.location.href = href;
          return;
        }
      };
    }

    return (
      <C
        {...cProps}
        ref={ref}
        css={[
          kind && kindCSS[kind],
          props.disabled && disabledCSS,
          props.size !== "default" && sizeCSS[props.size],
        ]}
        {...rest}
        onClick={onClick}
        className={`button ${props.className || ""}`}
      >
        {children}
      </C>
    );
  }
);

export const buttonCSS = css`
  border: var(--button-border-width) solid transparent;
  border-radius: var(--border-radius);
  padding: var(--button-padding);
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  text-decoration: none;
  text-align: center;
  transition: all 0.3s;
  white-space: nowrap;

  & + .button {
    margin-left: 16px;
  }

  // Icons
  svg:first-child {
    margin-right: 0.3em;
  }
  svg:last-child {
    margin-left: 0.3em;
  }
`;

const Link = styled.a`
  ${buttonCSS};
`;

const Button = styled.button`
  ${buttonCSS}
`;

const primaryCSS = css`
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: #fff;
  box-shadow: 0 5px 25px rgba(var(--primary-color-rgb), 0.6);

  &:hover {
    box-shadow: 0 5px 45px rgba(var(--primary-color-rgb), 0.8);
    transform: translateY(-2px);
  }
`;

const outlineCSS = css`
  border: var(--button-border-width) solid var(--stroke-color);
  color: var(--font-color-primary);

  &:hover {
    background-color: rgba(255, 255, 255, 0.1);
    /* border-color: var(--stroke-color-light); */
  }
`;

const outlinePrimaryCSS = css`
  border: var(--button-border-width) solid #b46ccd;

  background: linear-gradient(180deg, #5d5fef 0%, #ef5da8 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
`;

const outlineHighContrastCSS = css`
  ${outlineCSS}
  border-color: #fff;
`;

const blackCSS = css`
  border: var(--button-border-width) solid var(--black);
  background: var(--black);
  color: #fff;

  &:hover {
    ${primaryCSS};
  }
`;

const disabledCSS = css``;

const sizeCSS = {
  small: css`
    padding: var(--button-padding-small);
    font-size: 0.7rem;
  `,
  medium: css`
    padding: var(--button-padding-medium);
    font-size: 0.9rem;
  `,
};

const kindCSS: { [item in Kinds]: SerializedStyles } = {
  primary: primaryCSS,
  outline: outlineCSS,
  outlineHighContrast: outlineHighContrastCSS,
  outlinePrimary: outlinePrimaryCSS,
  black: blackCSS,
};
