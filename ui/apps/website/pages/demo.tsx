import { useEffect } from 'react';
import Head from 'next/head';
import styled from '@emotion/styled';

import Block from '../shared/legacy/Block';
import Button from '../shared/legacy/Button';
import Footer from '../shared/legacy/Footer';
import IconList from '../shared/legacy/IconList';
import Content from '../shared/legacy/content';
import Nav from '../shared/legacy/nav';

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: 'Product Demo Video',
        description: 'Learn how you can create, test and deploy functions in minutes',
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
          <Button
            size="medium"
            kind="primary"
            href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=demo-cta`}
          >
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
