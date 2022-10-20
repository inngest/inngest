import React, { useState } from "react";
import styled from "@emotion/styled";
import Logo from "../shared/Icons/Logo";
import Button from "../shared/Button";
import Hamburger from "../shared/Icons/Hamburger";
import Discord from "../shared/Icons/Discord";
import Github from "../shared/Icons/Github";

type Props = {
  nolinks?: boolean;
  sticky?: boolean;

  nodemo?: boolean;
};

const Nav: React.FC<Props> = (props) => {
  return (
    <NavWrapper sticky={props.sticky}>
      <NavContent {...props} />
    </NavWrapper>
  );
};

const NavContent: React.FC<Props> = (props: Props) => {
  const [show, setShow] = useState(false);
  return (
    <Container className={`max-w-5xl mx-auto ${show && "show"}`}>
      <div className="px-4 lg:px-0">
        {props.nolinks ? (
          <Logo width={115} className="logo" />
        ) : (
          <a href="/?ref=nav">
            <Logo width={115} className="logo" />
          </a>
        )}
      </div>

      {!props.nolinks && (
        <div className="links">
          <Hoverable>
            <StyledLink key="product" href="/features/sdk?ref=nav">
              Product
            </StyledLink>
            <div className="rounded-lg">
              <div className="primary-links pt-6 pb-2 px-2">
                <span className="text-slate-400 px-4">Product</span>
                <a href="/features/sdk?ref=nav" className="rounded-lg p-4">
                  <p>TypeScript & JavaScript SDK</p>
                  <p className="pt-1 text-slate-400">
                    <small>
                      Event-driven and scheduled serverless functions
                    </small>
                  </p>
                </a>
                <a
                  href="/features/step-functions?ref=nav"
                  className="rounded-lg p-4"
                >
                  <p>Step Functions</p>
                  <p className="pt-1 text-slate-400">
                    <small>Build complex conditional workflows</small>
                  </p>
                </a>
              </div>

              <div className="secondary-links bg-slate-100 pt-6 pb-3 px-2 text-xs rounded-r-lg">
                <span className="text-slate-400 px-4">Use cases</span>
                <a
                  href="/uses/serverless-cron-jobs?ref=nav"
                  className="rounded-lg px-4 py-3"
                >
                  Scheduled & cron jobs
                </a>
                <a
                  href="/uses/serverless-node-background-jobs?ref=nav"
                  className="rounded-lg px-4 py-3"
                >
                  Background tasks
                </a>
                <a
                  href="/uses/internal-tools?ref=nav"
                  className="rounded-lg px-4 py-3"
                >
                  Internal tools
                </a>
                <a
                  href="/uses/user-journey-automation?ref=nav"
                  className="rounded-lg px-4 py-3"
                >
                  User journey automation
                </a>
              </div>
            </div>
          </Hoverable>
          <Hoverable>
            <StyledLink key="product" href="/docs?ref=nav">
              Learn
            </StyledLink>
            <div className="rounded-lg">
              <div className="primary-links py-2 px-2">
                <a href="/docs?ref=nav" className="rounded-lg p-4">
                  <p>Docs</p>
                  <p className="pt-1 text-slate-400">
                    <small>
                      Everything you need to know about our event-driven
                      platform
                    </small>
                  </p>
                </a>
                <a href="/quick-starts?ref=nav" className="rounded-lg p-4">
                  <p>Quick starts</p>
                  <p className="pt-1 text-slate-400">
                    <small>
                      Example projects to reference when using Inngest
                    </small>
                  </p>
                </a>
              </div>
            </div>
          </Hoverable>
          <StyledLink key="blog" href="/blog?ref=nav">
            Blog
          </StyledLink>
          <StyledLink
            key="pricing"
            className="links-secondary"
            href="/pricing?ref=nav"
          >
            Pricing
          </StyledLink>
          <StyledLink key="discord" href={process.env.NEXT_PUBLIC_DISCORD_URL}>
            <Discord />
          </StyledLink>
          <StyledLink key="github" href="https://github.com/inngest/inngest-js">
            <Github />
          </StyledLink>
        </div>
      )}

      <div className="auth-options">
        <StyledLink
          className="auth-login"
          href="https://app.inngest.com/login?ref=nav"
        >
          Log in
        </StyledLink>
        {!props.nolinks && (
          <Button
            href="/sign-up?ref=nav"
            className="button"
            kind="primary"
            style={{ padding: "0.4rem 1rem" }}
          >
            Start building →
          </Button>
        )}
      </div>

      {!props.nolinks && (
        <a
          href="#"
          className="toggle"
          onClick={(e) => {
            e.preventDefault();
            setShow(!show);
          }}
        >
          <Hamburger size="24" />
        </a>
      )}
    </Container>
  );
};

const NavWrapper = styled.nav<{ sticky: boolean }>`
  position: ${({ sticky }) => (sticky ? "sticky" : "relative")};
  z-index: 100;
  top: ${({ sticky }) => (sticky ? "0" : "auto")};
  margin: 0 auto 1.5rem;
  background-color: var(--bg-color);
  box-shadow: 0 0 100px rgba(0, 0, 0, 0.07);
`;

const Container = styled.div<{ sticky?: boolean }>`
  position: relative;
  margin: 0 auto;
  z-index: 40;
  display: flex;
  padding: 1rem 1rem;
  max-width: 1280px;

  font-family: var(--font);
  font-size: 0.9em;

  .logo {
    // Offset for the g
    position: relative;
    top: 3px;
    z-index: 20;
  }

  svg {
    max-width: none; // fix reset.css issue on resize
  }

  > div,
  > a {
    /* Stack hamburger menu beneath the logo & toggle */
    position: relative;
    z-index: 2;
  }

  > div {
    display: flex;
    align-items: center;
  }

  .auth-options {
    gap: 1rem;
    margin-left: auto;
  }

  a:not(.button) {
    color: var(--font-color-primary);
  }

  .toggle {
    display: none;
  }

  @media (max-width: 1000px) {
    .links-secondary {
      display: none;
    }
  }

  @media (max-width: 920px) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    .auth-login {
      display: none;
    }
  }

  // Non-mobile nav
  @media (min-width: 800px) {
    .links {
      margin-left: 1.6rem;
      justify-content: start;
    }
  }
  // Mobile nav
  @media only screen and (max-width: 800px) {
    grid-template-columns: 1fr 64px;
    grid-column: 2 / -2;
    // unset

    .auth-options {
      display: none;
    }

    .links {
      /* Hide in a way that enables transitions */
      pointer-events: none;
      opacity: 0;
      transition: opacity 0.3s;

      /**
       * When shown, add a background to the entire menu by pos absolute
       * so that it sticks to the top of the page beneath the logo and hamburger
       * icon
       */
      position: absolute;
      background: var(--bg-color);
      padding-top: var(--nav-height);
      padding-bottom: 1rem;
      top: 0;
      left: 0;
      right: 0;
      z-index: 0;
      box-shadow: 0 0 40px rgba(var(--black-rgb), 0.8);

      /**
       * In order to maintain the same left-align as the logo, we need to transform
       * the link container into a grid with the same columns.
       */
      display: grid;
      grid-template-columns: repeat(10, 1fr);
      align-items: stretch;

      > a,
      > div {
        grid-column: 2 / -2;
      }
      > div {
        padding: 0;
      }
      a,
      > a {
        display: block;
        margin: 0;
        padding: 0.5rem 4px;
      }
    }

    align-items: center;

    .toggle {
      display: block;
      padding: 1rem;
    }

    &.show .links {
      opacity: 1;
      pointer-events: inherit;
    }
  }
`;

const StyledLink = styled.a`
  display: inline-flex;
  align-items: center;
  padding: 0.4rem 1rem 0.35rem;
  min-height: calc(1.5em + 0.3rem + 0.25rem); // make icons same height as text
  transition: all 0.2s;
  text-decoration: none;
  border-radius: var(--border-radius);
  white-space: nowrap;

  color: var(--font-color-primary);
  transition: all 0.3s;

  &:hover {
    background: #2f6d9d11;
  }
`;

const Hoverable = styled.div`
  position: relative;

  &.visible > div,
  &:hover > div {
    opacity: 1;
    pointer-events: all;
    transform: translateY(0);
    transition: all 0.3s;
  }

  > div {
    display: flex;
    flex-direction: column;
    white-space: nowrap;

    /* This keeps the hover focus in between the original button and this menu */
    &:before {
      content: "";
      display: block;
      background: transparent;
      height: 30px;
      top: -30px;
      left: 0;
      position: absolute;
      width: 100%;
    }

    opacity: 0;
    pointer-events: none;
    transform: translateY(20px);
    transition: all 0.3s;

    position: absolute;
    top: 60px;
    left: calc(-1.75rem + 20px);
    z-index: 3;
    background: var(--highlight-color);
    box-sizing: border-box;
    box-shadow: 0 0 100px rgba(0, 0, 0, 0.15), 0 10px 20px rgba(0, 0, 0, 0.08);
  }

  span {
    display: block;
    text-transform: uppercase;
    letter-spacing: 1.5px;
    font-size: 12px;
    line-height: 1.25;
    margin: 0 0 0.75rem;
  }

  // Completely remove submenus on small screens to prevent overflow and layout issues
  @media only screen and (max-width: 800px) {
    &:hover > div,
    > div {
      display: none;
    }
  }

  @media only screen and (min-width: 940px) {
    > div {
      flex-direction: row;
    }
  }

  .primary-links,
  .secondary-links {
    a {
      display: block;
      margin: 0;
      &:hover {
        background: #2f6d9d11;
      }
    }
    p {
      margin: 0;
      line-height: 1.05;
    }
  }
  .secondary-links a {
    color: var(--color-almost-black);
  }
`;

export default Nav;
