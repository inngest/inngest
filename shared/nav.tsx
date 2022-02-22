import React, { useState } from "react"
import styled from "@emotion/styled"
import Button from "../shared/Button"
import Hamburger from "../shared/Icons/Hamburger"

type Props = {
  nolinks?: boolean
}

const Nav: React.FC<Props> = (props) => {
  return (
    <div className="grid nav-grid">
      <NavContent {...props} />
      <div className="grid-line" />
    </div>
  )
}

const NavContent: React.FC<Props> = (props: Props) => {
  const [show, setShow] = useState(false)
  return (
    <Container className={["grid-center-8", show ? "show" : ""].join(" ")}>
      <div>
        <a href="/">
          <img src="/logo-white.svg" alt="Inngest logo" className="logo" />
        </a>
      </div>

      {!props.nolinks && (
        <>
          <div className="links">
            <StyledLink href="/library">Library</StyledLink>
            <StyledLink href="/docs">Docs</StyledLink>
            <StyledLink href="/blog">Blog</StyledLink>
            <StyledLink href="/pricing">Pricing</StyledLink>
          </div>
        </>
      )}

      <div>
        <StyledLink href="https://app.inngest.com/login">Log in</StyledLink>
        <Button
          href="/sign-up"
          className="button"
          kind="primary"
          style={{ padding: "0.4rem 0.8rem" }}
        >
          Start building →
        </Button>
      </div>

      <a href="#" className="toggle" onClick={() => setShow(!show)}>
        <Hamburger />
      </a>
    </Container>
  )
}

const Container = styled.div`
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2rem 0;
  height: 120px;

  font-family: var(--font);
  font-size: 22px;

  .button {
    font-weight: 600;
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

  img {
    max-height: 60px;
    margin: 5px 40px 0 4px;
  }

  a + a {
    margin-left: 5px;
  }

  a + a.button {
    margin-left: 20px;
  }

  .toggle {
    display: none;
  }

  @media only screen and (max-width: 950px) {
    div:last-of-type {
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

      a {
        grid-column: 2 / -2;
        margin: 0;
        padding: 0.5rem 4px;
      }
    }

    align-items: center;

    .toggle {
      display: block;
      padding: 1rem;
      margin-right: -1rem;
    }

    &.show .links {
      opacity: 1;
      pointer-events: inherit;
    }
  }
`

const StyledLink = styled.a`
  display: inline-block;
  padding: 12px 20px 11px;
  transition: all 0.2s;
  text-decoration: none;
  border-radius: var(--border-radius);

  &[href]:not([href=""]) {
    color: #fff;
  }

  &:hover {
    background: #2f6d9d11;
  }
`

const Hoverable = styled.div`
  position: relative;

  &:hover > div {
    opacity: 1;
    pointer-events: all;
    transform: translateY(0);
    transition: all 0.3s;
  }

  > div {
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
    width: 350px;
    top: 60px;
    left: calc(-1.75rem + 20px);
    z-index: 3;
    background: #fff;
    border-radius: 3px;
    box-shadow: 0 8px 50px rgba(0, 0, 0, 0.5);
    padding: 1.75rem;
    box-sizing: border-box;

    p {
      color: var(--light-grey);
      text-transform: uppercase;
      font-size: 0.8rem;
      margin: 2.5rem 0 1rem;
    }

    a {
      font-weight: 700;
      display: block;
      color: var(--blue-right) !important;
      text-decoration: none;
      transition: all 0.2s;

      &:hover,
      &:hover span {
        color: #fff;
      }
    }

    a + a {
      margin: 1.75rem 0 0 0;
    }

    a span {
      color: var(--light-grey);
      display: block;
      font-weight: 400;
      margin: 5px 0;
      transition: all 0.2s;
    }
  }
`

export default Nav
