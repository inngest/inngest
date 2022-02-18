import React from "react";
import styled from "@emotion/styled";

const Footer = () => {
  return (
    <Wrapper className="grid">
      <div className="grid-center-6">
        <div className="four-cols">
          <a href="https://www.inngest.com">
            <img
              src="/logo-white.svg"
              alt="Inngest logo"
              height="30"
              className="logo"
            />
          </a>
        </div>
        <div className="four-cols">
          <div>
            <p>Company</p>
            <a href="/about">About</a>
            <a href="/careers">Careers</a>
            <a href="/contact">Contact Us</a>
          </div>
          <div>
            <p>Product</p>
            <a href="/docs">Documentation</a>
            <a href="/integrations">Integrations</a>
            <a href="/docs/event-http-api-and-libraries">Libraries & SDKs</a>
          </div>
          <div>
            <p>Community</p>
            <a href="https://discord.gg/EuesV2ZSnX" rel="nofollow">
              Discord
            </a>
            <a href="https://github.com/inngest" rel="nofollow">
              Github
            </a>
            <a href="https://twitter.com/inngest" rel="nofollow">
              Twitter
            </a>
          </div>
          <div>
            <p>Legal</p>
            <a href="/privacy">Privacy</a>
            <a href="/terms">Terms and Conditions</a>
            <a href="/security">Security</a>
          </div>
        </div>
        <div className="four-cols">
          <small>© {new Date().getFullYear()} Inngest Inc</small>
        </div>
      </div>
      <div className="grid-line" />
    </Wrapper>
  );
};

export default Footer;

const Wrapper = styled.div`
  overflow: hidden;
  padding: 40vh 0 0;
  margin-top: -40vh;

  font-family: var(--font);
  font-size: 22px;

  background: url(/assets/footer-grid.svg) no-repeat right 10%;
  background-size: cover;

  > div {
    padding: 20vh 0 5vh;
  }

  p {
    font-weight: bold;
    font-size: 1.35rem;
  }

  small {
    opacity: 0.5;
    margin: 3vh 0 0;
  }

  .logo {
    margin: 0 0 3vh;
  }

  a {
    display: block;
    color: #fff !important;
    text-decoration: none;
    margin: 1rem 0;
  }

  .footer-grid {
    position: absolute;
    width: 100%;
    bottom: 0;
    opacity: 0.5;
    z-index: 0;
  }
`;
