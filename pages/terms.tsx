import styled from "@emotion/styled";
import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";

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
      <Header />

      <Container>
        <Content>
          <iframe src="https://www.iubenda.com/terms-and-conditions/26885259" />
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
    min-height: calc(100vh - 200px);
    margin: 50px 0;
  }

  @media only screen and (max-width: 800px) {
    padding: 0 20px;
  }
`;
