import React from "react";
import styled from "@emotion/styled";

const Nav = () => {
  return (
    <Container>
      <div>
        <img src="/logo-blue.svg" alt="Inngest logo" />
      </div>
      <div>
        <a href="https://docs.inngest.com/docs/intro" target="_blank">
          Documentation
        </a>
        <a href="https://app.inngest.com/login">Sign in</a>

        <a
          href="https://calendly.com/inngest-thb/30min"
          className="button"
          rel="nofollow"
          target="_blank"
        >
          Request a free demo
        </a>
      </div>
    </Container>
  );
};

const Content = styled.div`
  max-width: 1200px;
  margin: 0 auto;

  @media only screen and (max-width: 800px) {
    padding: 0 20px;
  }
`;

const Container = styled(Content)`
  height: 70px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  font-size: 0.9rem;

  img {
    max-height: 40px;
  }

  a:not(.button) {
    display: inline-block;
    padding: 12px 20px 11px;
    transition: all 0.2s;
    text-decoration: none;
  }
  a:not(.button):hover {
    background: #2f6d9d11;
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
