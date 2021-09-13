import Head from "next/head";
import styled from "@emotion/styled";
import Marquee from "react-fast-marquee";
import Nav from "../shared/nav";
import Footer from "../shared/footer";
import Content from "../shared/content";
import Tag, { greyCSS } from "../shared/tag";

export default function Product() {
  return (
    <>
      <Head>
        <title>
          Inngest → serverless event-driven & scheduled workflow automation
          platform for developers & operators
        </title>
        <link rel="icon" href="/favicon.png" />
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

      <Nav dark />

      <Hero className="hero-dark">
        <Content>
          <div>
            <h1>The modern event automation platform</h1>
            <p>
              Inngest lets you run real-time workflows in response to events,
              with zero infrastructure.
            </p>
            <p>
              We aggregate all events in your business, from internal and
              external systems, then allow you to build complex workflows to
              automate anything you need.
            </p>
          </div>
          <div>
            <img src="/screenshot-event.png" alt="An event within Inngest" />
            <img src="/screenshot-workflow.png" alt="An event within Inngest" />
          </div>
        </Content>
      </Hero>

      <Details>
        <Content>
          <h2 className="text-center">How it works</h2>

          <HIW>
            <SendBox>
              <p>Send us events</p>
              <p>
                <a href="https://docs.inngest.com/docs/events/sources/sources" target="_blank">Send us events</a> automatically via built-in integrations, our <a href="https://docs.inngest.com/docs/events/sources/api" target="_blank">APIs</a>, our <a href="https://docs.inngest.com/docs/events/sources/sdks" target="_blank">SDKs</a>,
                or&nbsp;<a href="https://docs.inngest.com/docs/events/sources/webhooks" target="_blank">webhooks</a>.
              </p>
            </SendBox>
            <SendBox>
              <p>Events are stored</p>
              <p>
                We process, transform, then store your events in your workspace,
                with full user and author information tracked
              </p>
            </SendBox>
            <SendBox>
              <p>Workflows are triggered</p>
              <p>
                Workflows run in real time when events are received, automating
                your processes with no manual input
              </p>
            </SendBox>
          </HIW>

          <Events>
            <small className="text-center">
              All events flow through Inngest, allowing you to automate anything
            </small>
            <EventTags
              speed={20}
              gradientColor={[243, 245, 245]}
              gradientWidth={60}
            >
              <Tag>payment received</Tag>
              <Tag>lead new</Tag>
              <Tag>stripe charge failed</Tag>
              <Tag>signup new</Tag>
              <Tag>appointment booked</Tag>
              <Tag>demo requested</Tag>
              <Tag>stripe invoice paid</Tag>
              <Tag>photo uploaded</Tag>
              <Tag>email received</Tag>
              <Tag>task completed</Tag>
            </EventTags>
          </Events>

          <h3 className="text-center">Powerful workflows</h3>

          <Grid>
            <div>
              <p>Drag &amp; Drop interface</p>
              <p>
                Visually build and understand workflows with the easy-to-use
                graph view.
              </p>
            </div>
            <div>
              <p>Complex logic supported</p>
              <p>
                Build out complex logic and branching, so that things run only
                when you want them to.
              </p>
            </div>
            <div>
              <p>Time management built-in</p>
              <p>
                Pause workflows anywhere, or wait until conditions are met to
                continue.
              </p>
            </div>
            <div>
              <p>Integrate with anything</p>
              <p>
                Easily integrate with your current tools to receive events and
                automate flows.
              </p>
            </div>
            <div>
              <p>Testing made easy</p>
              <p>
                Test mode comes built in, and you can run and debug workflows as
                you're building them.
              </p>
            </div>
            <div>
              <p>Serverless functions</p>
              <p>
                For full flexibility, run any code that you need within a
                workflow, in any language.
              </p>
            </div>
          </Grid>
        </Content>
      </Details>

      <Footer />
    </>
  );
}

const Hero = styled.div`
  padding: 100px 0 170px;
  font-size: 1.3125rem;

  h1 {
    font-size: 50px;
    font-size: 3.25rem;
  }

  > div {
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-gap: 80px;
  }

  div {
    position: relative;
  }

  img {
    height: auto;
    max-width: 100%;
    box-shadow: 0 0 20px rgba(0, 0, 0, 0.2);
    border-radius: 3px;
    position: absolute;
  }

  img:first-of-type {
    left: 0%;
  }

  img + img {
    max-width: 70%;
    position: absolute;
    top: 30%;
    left: -25px;
  }
`;

const Details = styled.div`
  background: linear-gradient(90deg, #f9f9f1 0%, #fbfbf6 100%);
  background: linear-gradient(
    180deg,
    rgba(243, 245, 245, 1) 20%,
    rgba(249, 251, 254, 1) 100%
  );
  box-shadow: inset 0 -20px 0 20px #fff;
  margin-top: -40px;
  margin-bottom: -20px;
  padding: 6rem 0 6rem;
  position: relative;

  h2 {
    margin: 0;
  }
`;

const HIW = styled.div`
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  grid-gap: 2rem;
  margin: 3.5rem 0 3rem;
  text-align: center;
  position: relative;

  > div {
    position: relative;
    z-index: 1;
  }
`;

const Events = styled.div`
  margin: 0 0 6rem;
  small {
    display: block;
    color: var(--light-grey);
  }
`;

const SendBox = styled.div`
  background: #fff;
  padding: 20px;
  box-shadow: 2px 5px 15px rgba(0, 0, 0, 0.08);

  display: flex;
  flex-direction: column;
  justify-content: center;

  p:first-of-type {
    font-weight: bold;
  }
  p {
    margin: 0.25rem 0;
  }
`;

const EventTags = styled(Marquee)`
  margin: 1rem 0 2rem;
  height: 1rem;
  overflow: hidden;

  span {
    display: inline-block;
    margin-left: 7px !important;
  }

  span {
    ${greyCSS};
    opacity: 0.8;
  }
`;

const Grid = styled.div`
  margin: 3rem 0;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-gap: 3rem 4rem;

  div p:first-of-type {
    font-weight: bold;
    font-size: 1.125rem;
  }
  div p {
    margin: 0.25rem 0;
  }
`;
