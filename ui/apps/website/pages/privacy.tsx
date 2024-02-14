import styled from "@emotion/styled";
import React from "react";
import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";
import { Button } from "src/shared/Button";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Privacy",
        description: "Inngest's privacy policy",
      },
    },
  };
}

export default function Privacy() {
  return (
    <>
      <Header />

      <Container>
        <Content>
          <iframe src="https://www.iubenda.com/privacy-policy/26885259" />
        </Content>
      </Container>

      <Footer />
    </>
  );
}

const Content = styled.div`
  max-width: 1200px;
  margin: 0 auto;

  iframe {
    border: 0;
    width: 100%;
    min-height: 1200px;
    margin: 50px 0;
  }

  @media only screen and (max-width: 800px) {
    padding: 0 20px;
  }
`;
