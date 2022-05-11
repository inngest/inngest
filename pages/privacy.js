import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/footer";
import Nav from "../shared/nav";

export default function Home() {
  return (
    <>
      <Head>
        <title>
          Inngest → serverless event-driven & scheduled workflow automation
          platform for developers & operators
        </title>
        <meta property="og:title" content="Inngest" />
        <meta property="og:url" content="https://www.inngest.com" />
        <meta property="og:image" content="/logo.svg" />
        <meta
          property="og:description"
          content="Build, run, operate, and analyze your workflows in minutes."
        />
        <script src="/inngest-sdk.js"></script>
        <script
          defer
          src="https://static.cloudflareinsights.com/beacon.min.js"
          data-cf-beacon='{"token": "e2fa9f28c34844e4a0d29351b8730579"}'
        ></script>
      </Head>

      <Nav />

      <Content>
        <iframe src="https://www.iubenda.com/privacy-policy/26885259" />
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
