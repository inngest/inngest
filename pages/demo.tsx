import { useEffect } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Content from "../shared/content";

import Block from "../shared/Block";
import IconList from "../shared/IconList";
import Button from "../shared/Button";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Pricing",
        description: "Simple pricing. Powerful functionality.",
      },
    },
  };
}

export default function Demo() {
  return (
    <>
      <Nav />

      <DemoVideo>
        <iframe
          src="https://www.youtube.com/embed/qVXzYBcJmGU?autoplay=1"
          title="Inngest Product Demo"
          frameBorder="0"
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
          allowFullScreen
        ></iframe>
      </DemoVideo>

      <Hero>
        <h1>Create, Test, and Deploy in minutes</h1>
        <div className="hero-ctas">
          <Button size="medium" kind="primary" href="/sign-up?ref=demo-cta">
            Sign up for free
          </Button>
        </div>
      </Hero>

      <div style={{ marginTop: 100 }}>
        <Footer />
      </div>
    </>
  );
}

const Hero = styled.div`
  padding: calc(var(--nav-height) + 10vh) 0 10vh;
  margin-top: calc(var(--nav-height) * -1);
  text-align: center;

  h1 + p {
    font-size: 22px;
    line-height: 1.45;
    opacity: 0.8;
  }
  p {
    font-family: var(--font);
  }

  .hero-ctas {
    margin-top: 2em;
    display: flex;
    justify-content: center;
  }
`;

const DemoVideo = styled.div`
  margin: 10vh auto;

  // to make the video responsive
  position: relative;
  overflow: hidden;
  width: 90%;
  padding-top: 50.625%; // 16:9 ratio (90/16*9)
  text-align: center;

  iframe {
    position: absolute;
    top: 0;
    left: 0;
    bottom: 0;
    right: 0;
    width: 100%;
    height: 100%;
  }
`;
