import React from "react";
import styled from "@emotion/styled";

import Logo from "../Icons/Logo";
import Discord from "../Icons/Discord";
import Github from "../Icons/Github";

const Footer = () => {
  return (
    <Wrapper className="reg-grid">
      <div className="grid-center-8">
        <div className="four-cols">
          <a href="https://www.inngest.com">
            <Logo height={30} />
          </a>
        </div>
        <div className="four-cols">
          <div>
            <p>Product</p>
            <a href="/features/sdk?ref=footer">Function SDK</a>
            <a href="/features/step-functions?ref=footer">Step Functions</a>
            <a href="/docs?ref=footer">Documentation</a>
            <a href="/patterns?ref=footer">Patterns: Async + Event-Driven</a>
            <a
              href="https://typedwebhook.tools?ref=inngest-footer"
              className="typedwebhook-button"
            >
              TypedWebhook.tools
            </a>
          </div>
          <div>
            <p>Use Cases</p>
            <a href="/uses/serverless-node-background-jobs?ref=footer">
              Node.js background jobs
            </a>
            <a href="/uses/internal-tools?ref=footer">Internal tools</a>
            <a href="/uses/user-journey-automation?ref=footer">
              User Journey Automation
            </a>
          </div>
          <div>
            <p>Company</p>
            <a href="/about">About</a>
            <a href="/blog">Blog</a>
            <a href="/careers">Careers</a>
            <a href="/contact">Contact Us</a>
            <a href={process.env.NEXT_PUBLIC_SUPPORT_URL}>Support</a>
          </div>
          <div>
            <p>Community</p>
            <a href={process.env.NEXT_PUBLIC_DISCORD_URL} rel="nofollow">
              <Discord /> Discord
            </a>
            <a href="https://github.com/inngest/inngest-js" rel="nofollow">
              <Github /> Github
            </a>
            <a href="https://twitter.com/inngest" rel="nofollow">
              Twitter
            </a>
          </div>
          <div></div>
        </div>
        <div className="footer-small-print flex flex-column gap-4">
          <div>© {new Date().getFullYear()} Inngest Inc</div>
          <a href="/privacy">Privacy</a>
          <a href="/terms">Terms and Conditions</a>
          <a href="/security">Security</a>
        </div>
      </div>
    </Wrapper>
  );
};

export default Footer;

const Wrapper = styled.div`
  overflow: hidden;
  padding: 40vh 0 0;
  margin-top: -40vh;

  font-family: var(--font);
  font-size: 0.9rem;

  background: url(/assets/footer-grid.svg) no-repeat right 10%;
  background-size: cover;

  > div {
    padding: 20vh 0 5vh;
  }

  p {
    font-weight: bold;
    font-size: 1rem;
  }

  .footer-small-print {
    margin-top: 3vh;
    font-size: 0.8em;
    color: var(--font-color-secondary);

    a {
      margin: 0;
      color: var(--font-color-secondary);
    }
  }

  .logo {
    margin: 0 0 3vh;
  }

  a {
    display: flex;
    align-items: center;
    color: var(--font-color-primary);
    text-decoration: none;
    margin: 1rem 0;

    // Icons
    svg {
      margin-right: 0.4rem;
    }
  }

  .footer-grid {
    position: absolute;
    width: 100%;
    bottom: 0;
    opacity: 0.5;
    z-index: 0;
  }

  .typedwebhook-button {
    display: inline-block;
    margin: 0.5rem 0;
    padding: 0.3rem 0.5rem;

    background-color: var(--primary-color);
    background: radial-gradient(
        62.5% 62.5% at 20.5% 95.25%,
        rgba(254, 255, 191, 0.25) 0%,
        rgba(255, 237, 191, 0.035) 100%
      ),
      radial-gradient(
        72% 92.13% at 78.88% 16.5%,
        rgba(181, 81, 198, 0.3243) 0%,
        rgba(124, 87, 128, 0.0893) 100%
      ),
      linear-gradient(180deg, #4636f5 0%, #1d66d2 100%);
    color: var(--color-white);
    font-family: var(--font-mono);
    font-size: 0.7rem;
    font-weight: bold;
    border-radius: var(--border-radius);
    transition: all 200ms ease-in-out;

    &:hover {
      box-shadow: 0 5px 45px rgba(var(--primary-color-rgb), 0.6);
      transform: translateY(-0.1rem);
    }
  }

  @media (max-width: 400px) {
    margin-left: 1em;
    .grid-center-6 {
      grid-column: 2/-2;
    }
    > div {
      padding: 10vh 0 5vh;
    }
  }
`;
