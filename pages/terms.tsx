import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/Footer";
import Nav from "../shared/nav";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Terms",
        description: "Inngest's terms and conditions",
      },
    },
  };
}

export default function Terms() {
  return (
    <>
      <Nav />

      <Content>
        <iframe src="https://www.iubenda.com/terms-and-conditions/26885259" />
      </Content>

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
    min-height: calc(100vh - 200px);
    margin: 50px 0;
  }

  @media only screen and (max-width: 800px) {
    padding: 0 20px;
  }
`;
