import React from "react";
import styled from "@emotion/styled";

const Footer = () => {
  return (
    <Container>
      <Content>
        <Links>
          <div>
            <a href="https://www.inngest.com">
              <img src="/logo-white.svg" alt="Inngest logo" height="30" />
            </a>
            <small>© 2021 Inngest Inc</small>
          </div>
          <div>
            <strong>Inngest</strong>
            <a href="https://www.inngest.com">Product</a>
            <a href="https://docs.inngest.com">Documentation</a>
            <a href="https://www.inngest.com/security">Security</a>
          </div>
          <div>
            <strong>Community</strong>
            <a href="https://discord.gg/EuesV2ZSnX">Discord</a>
            <a href="https://twitter.com/inngest">Twitter</a>
          </div>
        </Links>
      </Content>
    </Container>
  );
};

const Links = styled.div`
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  font-size: 14px;
  line-height: 1.8;
  opacity: 0.7;

  img {
    display: block;
    margin: 0 0 4px;
  }

  strong {
    display: block;
    margin-bottom: 10px;
  }

  a {
    display: block;
    color: #fff;
  }

  @media only screen and (max-width: 800px) {
    display: flex;
    flex-flow: column;
    align-items: center;
    text-align: center;

    strong {
      margin-top: 20px;
    }
  }
`;

const Container = styled.div`
  margin-top: 100px;
  background: #222631;
  color: #fff;
  padding: 40px 20px;
  font-size: 0.9rem;
`;

const Content = styled.div`
  max-width: 1200px;
  margin: 0 auto;
`;

export default Footer;
