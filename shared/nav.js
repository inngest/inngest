import React from "react";
import styled from "@emotion/styled";

const Nav = ({ dark }) => {
  return (
    <div className={dark ? "nav-dark" : ""}>
      <Container>
        <div>
          <a href="https://www.inngest.com">
            {dark ? (
              <img src="/logo-white.svg" alt="Inngest logo" />
            ) : (
              <img src="/logo-blue.svg" alt="Inngest logo" />
            )}
          </a>
          <Hoverable>
            <StyledLink href="/product">Product</StyledLink>
            <div>
              <a href="/product">
                How it works
                <span>An overview to the platform</span>
              </a>
              <a href="/product">
                Features
                <span>Things that make Inngest unique</span>
              </a>

              <p>Inngest for...</p>
              <a href="/product/for-product">
                Product
                <span>Rapid development and iteration</span>
              </a>
              <a href="/product/for-operations">
                Operations
                <span>Simple management and full visibility</span>
              </a>
              <a href="/product/for-engineering">
                Engineering
                <span>Frictionless setup and handoff</span>
              </a>
            </div>
          </Hoverable>
          <StyledLink href="/pricing">Pricing</StyledLink>
          <Hoverable>
            <StyledLink href="/company">Company</StyledLink>
            <div>
              <a href="/company">About us</a>
              <a href="/company/contact">Contact us</a>
              <a href="/company/press">Press</a>
            </div>
          </Hoverable>
          <StyledLink
            href="https://docs.inngest.com/docs/intro"
            target="_blank"
          >
            Docs
          </StyledLink>
        </div>

        <div>
          <StyledLink href="https://app.inngest.com/login">Sign in</StyledLink>

          <a
            href="https://calendly.com/inngest-thb/30min"
            className="button"
            rel="nofollow"
            target="_blank"
          >
            See how it works
          </a>
        </div>
      </Container>
    </div>
  );
};

const Content = styled.div`
  max-width: 1200px;
  margin: 0 auto;

  @media only screen and (max-width: 800px) {
    padding: 0 20px;
  }
`;

const StyledLink = styled.a`
  display: inline-block;
  padding: 12px 20px 11px;
  transition: all 0.2s;
  text-decoration: none;

  &:hover {
    background: #2f6d9d11;
  }
`;

const Hoverable = styled.div`
  position: relative;

  &:hover > div {
    opacity: 1;
    pointer-events: all;
    transform: translateY(0);
    transition: all .3s;
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
    transition: all .3s;

    position: absolute;
    width: 350px;
    top: 60px;
    left: calc(-1.75rem + 20px);
    z-index: 1;
    background: #fff;
    border-radius: 3px;
    box-shadow: 0 8px 50px rgba(0, 0, 0, 0.5);
    padding: 1.75rem;
    box-sizing: border-box;

    p {
      color: var(--light-grey);
      text-transform: uppercase;
      font-size: .8rem;
      margin: 2.5rem 0 1rem;
    }

    a {
      font-weight: 700;
      display: block;
      color: var(--blue-right);
      text-decoration: none;
      transition: all .2s;

      &:hover, &:hover span {
        color: var(--blue-left);
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
      transition: all .2s;
    }

    a +
  }
`;

const Container = styled(Content)`
  height: 70px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  font-size: 0.9rem;
  font-weight: 600;

  > div {
    display: flex;
    align-items: center;
  }

  img {
    max-height: 60px;
    margin: 5px 40px 0 0;
  }

  a + a {
    margin-left: 5px;
  }

  a + a.button {
    margin-left: 20px;
  }

  @media only screen and (max-width: 800px) {
    div:last-of-type {
      display: none;
    }
  }
`;

export default Nav;
