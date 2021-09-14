import Head from "next/head";
import styled from "@emotion/styled";
import Marquee from "react-fast-marquee";
import Nav from "../shared/nav";
import Footer from "../shared/footer";
import Content from "../shared/content";
import Action, { Outline } from "../shared/action";
import Tag, { greyCSS } from "../shared/tag";
import DragFC from "../shared/drag";

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
              We aggregate all events in your business from internal and
              external systems, then allow you to build complex workflows which
              react to events &mdash; automating anything you need.
            </p>
          </div>
          <div className="images">
            <img src="/screenshot-event.png" alt="An event within Inngest" />
            <img src="/screenshot-workflow.png" alt="An event within Inngest" />
          </div>
        </Content>
      </Hero>

      <How>
        <Content>
          <h2 className="text-center">How it works</h2>

          <HIW>
            <SendBox>
              <p>Send us events</p>
              <p>
                <a
                  href="https://docs.inngest.com/docs/events/sources/sources"
                  target="_blank"
                >
                  Send us events
                </a>{" "}
                automatically via built-in integrations, our{" "}
                <a
                  href="https://docs.inngest.com/docs/events/sources/api"
                  target="_blank"
                >
                  APIs
                </a>
                , our{" "}
                <a
                  href="https://docs.inngest.com/docs/events/sources/sdks"
                  target="_blank"
                >
                  SDKs
                </a>
                , or&nbsp;
                <a
                  href="https://docs.inngest.com/docs/events/sources/webhooks"
                  target="_blank"
                >
                  webhooks
                </a>
                .
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
        </Content>
      </How>

      <Details>
        <Content>
          <a name="features">
            <h3 className="text-center">The most powerful workflow engine</h3>
          </a>
          <p className="subtitle text-center">
            Our powerful workflow engine allows complex, multi-step workflows
            which can coordinate between events in your business.
            <br />
            For engineers, workflows are defined with a strongly typed config
            language. For operators, workflows can be edited visually.
          </p>

          <Grid style={{ paddingTop: 20 }}>
            <div>
              <Drag>
                <DragFC
                  name="Create lead in Salesforce"
                  subtitle="From the account in the event"
                  icon="/icons/sf-cloud.svg"
                  cursor
                />
              </Drag>

              <p className="title">Drag &amp; Drop interface</p>
              <p>
                Visually build workflows to create integrations, define logic,
                and map data between apps.
              </p>
            </div>

            <div>
              <Logic>
                <span />
                <div className="white-tag">Value</div>
                <div className="white-tag">&gt;=</div>
                <div className="white-tag">7,500</div>
                <span />
              </Logic>
              <p className="title">Complex logic supported</p>
              <p>
                Build out complex logic and branching, so that things run only
                when you want them to.
              </p>
            </div>
            <div>
              <Time>
                <div className="white-tag">
                  <img src="/icons/clock.svg" /> Wait <b>3 days</b>
                </div>
                <div className="white-tag">
                  <img src="/icons/clock.svg" /> Wait for{" "}
                  <b>invoice end date</b>
                </div>
                <div className="white-tag">
                  <img src="/icons/clock.svg" /> If{" "}
                  <b>email bounces within 1 day</b>
                </div>
              </Time>
              <p className="title">Time management built-in</p>
              <p>
                Pause workflows anywhere, or wait until conditions are met to
                continue.
              </p>
            </div>
            <div>
              <Logos className="text-center">
                <div>
                  <img src="/integrations/salesforce.png" alt="Salesforce" />
                  <img
                    src="/integrations/clickup.png"
                    alt="Clickup"
                    height="25"
                  />
                  <img src="/integrations/stripe.png" alt="Stripe" />
                  <img src="/integrations/onesignal.png" alt="One Signal" />
                  <img src="/integrations/slack.png" alt="Slack" />
                  <img src="/integrations/twilio.png" alt="Twilio" />
                </div>
              </Logos>
              <p className="title">Integrate with anything</p>
              <p>
                Easily integrate with your current tools to receive events and
                automate flows.
              </p>
            </div>

            <div>
              <GridGraphic>
                <img
                  src="/assets/step-over.svg"
                  style={{ marginRight: 20, opacity: 0.5 }}
                />
                <div className="white-tag">Run next action: sync to Jira</div>
              </GridGraphic>
              <p className="title">Testing made easy</p>
              <p>
                Test mode comes built in, and you can run and debug workflows as
                you're building them.
              </p>
            </div>

            <div>
              <GridGraphic>
                <img
                  src="/assets/code.svg"
                  style={{ marginRight: 20, opacity: 0.5 }}
                />
                <div className="white-tag">
                  export default (evt, actions) => {"{"}
                  <br />
                  &nbsp;{" "}
                  <span style={{ opacity: 0.5 }}>// run any language</span>
                  <br />
                  {"}"}
                </div>
              </GridGraphic>
              <p className="title">Serverless functions</p>
              <p>
                For full flexibility, run any code in a workflow &mdash; in any
                language.{" "}
                <a
                  href="https://docs.inngest.com/docs/actions/serverless/tutorial"
                  target="_blank"
                >
                  View the docs
                </a>
                .
              </p>
            </div>
          </Grid>
        </Content>
      </Details>

      <Details>
        <Content>
          <Half>
            <div>
              <h3>Built for reliability and security</h3>
              <p>
                Ensure your workflows run smoothly every time, without manual
                steps. Plus, with data encrypted using custom data encryption
                keys you can rest assured your data is secure.
              </p>

              <ul>
                <li>
                  <b>Automatic retries</b> prevents issues when other services
                  are down
                </li>
                <li>
                  <b>Built-in logging</b> shows detailed workflow logs for each
                  step of a workflow
                </li>
                <li>
                  <b>Error handling</b> allows you to configure custom error
                  workflows any time issues occur
                </li>
              </ul>
            </div>
            <div>
              <img
                src="/screenshot-logs-shadow.png"
                alt="Workflow logs"
                className="no-shadow"
              />
            </div>
          </Half>
        </Content>
      </Details>

      <Details>
        <Content>
          <Half>
            <div>
              <img
                src="/screenshot-versioning.png"
                alt="Workflow versioning"
                className="shadow"
              />
            </div>
            <div>
              <h3>Made for rapid iteration across teams</h3>
              <p>
                Workflows are fully versioned with every change stored, and each
                workflow run stores detailed author information, allowing you to
                figure out which workflows each of your users run through.
              </p>

              <ul>
                <li>
                  <b>Unlimited users</b> for wide collaboration and visibility
                  within your company
                </li>
                <li>
                  <b>Version history</b> allows you to track every change from
                  the very start
                </li>
                <li>
                  <b>Workflow authoring</b> tracks which users run through which
                  workflows
                </li>
                <li>
                  <b>Drafts</b> allow you to prep and schedule workflow releases
                  easily
                </li>
              </ul>
            </div>
          </Half>
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

  .images {
    transform: perspective(1500px) rotateY(-15deg);
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

const How = styled.div`
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
  margin: 0;
  small {
    display: block;
    color: var(--light-grey);
  }
`;

const SendBox = styled.div`
  background: #fff;
  padding: 20px;
  border-radius: 3px;
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
  grid-gap: 5rem 4rem;

  .title {
    font-weight: bold;
    font-size: 1.125rem;
  }
  > div > p {
    margin: 0.25rem 0;
  }
`;

const GridGraphic = styled.div`
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 0 0 1.5rem;
  height: 100px;

  .white-tag {
    position: relative;
    box-sizing: border-box;
    border-radius: 5px;
    background: #fff;
    box-shadow: 0 3px 8px rgba(0, 0, 0, 0.05);
    padding: 7px 14px;
    font-size: 13px;
    border: 1px solid #e8e8e6;
    font-family: monospace;
    color: #777;
  }

  .white-tag b {
    font-weight: 600;
  }
`;

const Drag = styled(GridGraphic)`
  > div {
    position: relative;

    > img {
      width: 24px;
      height: 24px;
      position: absolute;
      z-index: 2;
      right: 6px;
      top: 7px;
      pointer-events: none;
    }

    .drop {
      position: absolute;
      top: 10px;
      left: 10px;
      z-index: 0;
    }
  }
`;

const Logic = styled(GridGraphic)`
  > span:first-of-type {
    display: block;
    width: 30px;
    height: 14px;
    content: "";
    background: url(/assets/if-left.svg) no-repeat;
    margin: 0 5px;
  }

  > span:last-of-type {
    display: block;
    width: 58px;
    height: 56px;
    content: "";
    background: url(/assets/if-right.svg) no-repeat;
    margin: 0 5px;
  }

  > div + div {
    margin-left: 7px;
  }
`;

const Time = styled(GridGraphic)`
  flex-direction: column;
  align-items: flex-start;

  > div {
    margin: 0 0 0.25rem 2rem;
    font-size: 13px !important;
  }
  > div:first-of-type {
    margin-left: 0;
  }
  > div:last-of-type {
    margin-left: 4rem;
  }

  img {
    height: 12px;
    margin: 0 3px -2px;
    opacity: 0.4;
  }
`;

const Logos = styled(GridGraphic)`
  overflow: hidden;
  align-items: center;
  justify-content: center;
  img {
    max-height: 28px;
    filter: grayscale(100%);
    opacity: 0.6;
    margin: 0.5rem 0.5rem;
  }
`;

const Half = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-gap: 80px;
  align-items: center;

  h3 {
    margin-top: 0;
  }

  img {
    object-fit: cover;
    width: 100%;
    max-height: 100%;
    border-radius: 10px;
  }

  img.shadow {
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
  }
  img.no-shadow {
    width: calc(100% + 40px);
    margin-left: -20px;
  }

  ul {
    list-style-type: none;
    margin: 2rem 0 0;
    padding: 0;
  }
`;
