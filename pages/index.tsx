import { useEffect, useState } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Button from "../shared/Button";
import Code from "../shared/Code";
import Callout from "../shared/Callout";
import Integration, { IntegrationType } from "../shared/Integration";
// Icons
import Hub from "../shared/Icons/Hub";
import Functions from "../shared/Icons/Functions";
import History from "../shared/Icons/History";
import Create from "../shared/Icons/Create";
import Deploy from "../shared/Icons/Deploy";
import Monitor from "../shared/Icons/Monitor";
import CLI from "../shared/Icons/CLI";
import Retries from "../shared/Icons/Retries";
import Replays from "../shared/Icons/Replays";
import Transforms from "../shared/Icons/Transforms";
import Versions from "../shared/Icons/Versions";
import Rollback from "../shared/Icons/Rollback";
import Audit from "../shared/Icons/Audit";
import Logging from "../shared/Icons/Logging";
import Historical from "../shared/Icons/Historical";
import Observability from "../shared/Icons/Observability";
import Alerting from "../shared/Icons/Alerting";
import { CheckBanner } from "../shared/Banner";

// Send event preformatted code
const events = {
  cURL: `curl -X POST "https://inn.gs/e/test-key-goes-here-bjm8xj6nji0vzzu0l1k" \\
  -d '{"name": "test.event", "data": { "email": "gob@bluth-dev.com" } }'`,
  JavaScript: `Inngest.event({
  name: "test.event",
  data: { email: "gob@bluth-dev.com" },
  user: { email: "gob@bluth-dev.com" },
});`,
  Go: `package main

import (
	"context"
	"os"

	"github.com/inngest/inngestgo"
)

func SendEvent(ctx context.Context) {
	// Create a new client
	client := inngestgo.NewClient(os.Getenv("INGEST_KEY"))
	// Send an event
	client.Send(ctx, inngestgo.Event{
		Name: "user.created",
		Data: map[string]interface{}{
			"plan": "pro",
			"ip":   "10.0.0.10",
		},
		User: map[string]interface{}{
			// Use the external_id field within User so that we can add context
			// for audit trails.
			inngestgo.ExternalID: user.ID,
			inngestgo.Email:      user.Email,
		},
		Version:   "2022-01-01.01",
		Timestamp: inngestgo.Now(),
	})
}`,
};

const integrations = [
  {
    name: "Stripe",
    logo: "/integrations/stripe.svg",
    category: "Payments & Billing",
    type: [IntegrationType.EVENTS],
  },
  {
    name: "Twilio",
    logo: "/integrations/twilio.svg",
    category: "Messaging & Communication",
    type: [IntegrationType.EVENTS],
  },
  {
    name: "Mailchimp",
    logo: "/integrations/mailchimp.svg",
    category: "Messaging & Communication",
    type: [IntegrationType.EVENTS],
  },
  {
    name: "Salesforce",
    logo: "/integrations/salesforce.svg",
    category: "Sales Enablement",
    type: [IntegrationType.EVENTS],
  },
  {
    name: "Chatwoot",
    logo: "/integrations/chatwoot.svg",
    category: "Customer support",
    type: [IntegrationType.EVENTS],
  },
  {
    name: "GitHub",
    logo: "/integrations/github.svg",
    category: "Software Collaboration",
    type: [IntegrationType.EVENTS],
  },
];

const SDKs = [
  {
    name: "JavaScript",
    logo: "/assets/languages/icon-logo-js.svg",
    url: "https://github.com/inngest/javascript-sdk",
  },
  {
    name: "Python",
    logo: "/assets/languages/icon-logo-python.svg",
    url: "https://github.com/inngest/inngest-python",
  },
  {
    name: "Go",
    logo: "/assets/languages/icon-logo-go.svg",
    url: "https://github.com/inngest/inngestgo",
  },
  {
    name: "Ruby",
    logo: "/assets/languages/icon-logo-ruby.svg",
    url: "https://github.com/inngest/inngest-ruby",
  },
];

const SectionHeader: React.FC<{
  label?: string;
  title: string;
  subtitle: string;
  counter?: string;
}> = ({ label, title, subtitle, counter }) => {
  return (
    <div className="grid section-header">
      <div className="grid-center-6 sm-col-8-center">
        <span className="section-label">{label}</span>
        <h2>{title}</h2>
        <h4>{subtitle}</h4>
      </div>
      <div className="grid-line">{counter && <span>{counter}</span>}</div>
    </div>
  );
};

// TODO: move these into env vars
export const INGEST_KEY =
  "BIjxBrM6URqxAu0XgIAae5HgBCv8l_LodmdGonFCfngjhwIgQEbvbUUQTwvFMHO21vxCJEGsC7KPdXEzdXgOAQ";

// test key
// export const INGEST_KEY = 'MnzaTCk7Se8i74hA141bZGS-NY9P39RSzYFbxanIHyV2VDNu1fwrns2xBQCEGdIb9XRPtzbp0zdRPjtnA1APTQ';

export default function Home() {
  useEffect(() => {
    // Defer loading of background textures.
    const style = document.createElement("style");
    style.innerText = `
      .home { 
        background: url(/assets/texture.webp) repeat-y;
        background-size: cover;
      }
    `;
    style.id = "bg-texture";
    document.body.appendChild(style);
    return () => {
      document.querySelector("#bg-texture").remove();
    };
  }, []);

  return (
    <Wrapper className="home">
      <Head>
        <title>
          Inngest → build serverless event-driven functions in minutes
        </title>
        <link rel="icon" href="/favicon.png" />
        <meta
          property="og:title"
          content="Inngest - build serverless event-driven functions in minutes"
        />
        <meta
          property="og:description"
          content="Create, deploy, and monitor event-driven serverless functions with confidence."
        />
        <meta property="og:url" content="https://www.inngest.com" />
        <meta property="og:image" content="/logo.svg" />
        <meta
          property="og:description"
          content="Build event serverless event-driven systems in seconds"
        />
        <link
          rel="stylesheet"
          href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.4.0/styles/github-dark.min.css"
        />
        <script src="/inngest-sdk.js" defer async></script>
        <script
          defer
          src="https://static.cloudflareinsights.com/beacon.min.js"
          data-cf-beacon='{"token": "e2fa9f28c34844e4a0d29351b8730579"}'
        ></script>
      </Head>

      <Nav />

      <div className="grid hero-grid">
        <Hero className="grid-center-8">
          <div>
            <h1>Make event-driven apps fun to build</h1>
            <p className="subheading">
              Deploy serverless functions in minutes.
              <br />
              No infra. No servers. Zero YAML.
            </p>

            <Button kind="primary" href="/sign-up">
              <span className="button-text-light">{">"}_</span>
              &nbsp;
              <span className="button-text-med">Start building</span>
            </Button>
            <Button kind="outline" href="/docs">
              Explore docs →
            </Button>
          </div>
          <img src="/assets/preview.svg" alt="Inngest visualization" />
        </Hero>
        <div className="grid-line" />
      </div>
      <CheckBanner
        className="monospace"
        list={[
          "Developer CLI",
          "Auto-gen'd types & schemas",
          "Retries & replays built in",
        ]}
      />

      <SectionHeader
        label="Introducing our"
        title="Event mesh"
        subtitle="Everything you need to build production ready event driven apps."
        counter="/01"
      />

      <div className="grid">
        <div className="grid-center-6 sm-col-8-center">
          <img
            src="/assets/graphic.svg"
            alt="Event driven serverless function example"
            className="full-width img-mesh"
          />
        </div>
        <div className="grid-line" />
      </div>

      <SectionHeader
        title="How it works"
        subtitle="Our Event Mesh makes it easy to build event-driven apps."
        counter="/02"
      />

      <HIW className="grid">
        <div className="grid-2-offset-2 sm-col-8-center">
          <Hub />
          <h3>One event hub</h3>
          <p>
            We ingest all your events via our one-click integrations, SDKs, or
            webhooks.
          </p>
        </div>
        <div className="grid-2 sm-col-8-center">
          <Functions />
          <h3>Serverless Functions</h3>
          <p>
            Your code is executed instantly against the events you specify.
            Automatic retries built-in.
          </p>
        </div>
        <div className="grid-2 sm-col-8-center">
          <History />
          <h3>Unified History</h3>
          <p>
            View logging, payload data, and audit-trails for your events and
            functions together in one place.
          </p>
        </div>
        <div className="grid-line" />
      </HIW>

      <SectionHeader
        label="Events"
        title="Send your events from anywhere"
        subtitle="Use our SDKs or webhooks to send events from your app"
        counter="/03"
      />

      <div className="section code-grid grid">
        <div className="code grid-center-6 sm-col-8-center">
          <Code code={events} />
          <div className="sdk-list">
            <p className="text-center">Get started with an SDK: </p>
            {SDKs.map((sdk, idx) => (
              <a key={idx} href={sdk.url} className="sdk-list-item">
                <img src={sdk.logo} alt={`${sdk.name} SDK`} />
              </a>
            ))}
          </div>
        </div>

        <div className="grid-line" />
      </div>

      <div className="section grid">
        <div className="grid-center-6 sm-col-8-center">
          <h4>
            Automatically stream events from 3rd party apps with our
            integrations
          </h4>
          <div className="integrations">
            {integrations.map((i) => (
              <Integration {...i} key={i.name} />
            ))}
          </div>
        </div>
        <div className="grid-line" />
      </div>

      <SectionHeader
        label="DX First"
        title="Build with superpowers"
        subtitle="Create, deploy, and monitor event-driven serverless functions with confidence."
        counter="/04"
      />

      <div className="grid">
        <div className="grid-center-6 sm-col-8-center two-cols dx">
          <div>
            <Create />
            <h3>Create and test with real data</h3>
            <p>
              Start building functions with our auto-typed example payloads or
              use historical event data. Easily run fuzz testing and handle type
              changes without issues.
            </p>
          </div>
          <img src="/assets/payload.svg" alt="Event payload" />

          <div>
            <Deploy />
            <h3>Deploy with confidence</h3>
            <p>
              Get realtime insights into which payloads are causing errors.
              Instantly rollback to any previous version. And replay failed
              payloads when an issue is resolved.{" "}
            </p>
          </div>
          <img src="/assets/deploy.svg" alt="Deployed functions and events" />

          <div>
            <Monitor />
            <h3>Monitor your serverless functions</h3>
            <p>
              Get granular visibility into event → function pathways including
              conditional execution, function chains, and how often each
              function runs.
            </p>
          </div>
          <img src="/assets/history.svg" alt="Deployed function history" />
        </div>

        <div className="grid-line" />
      </div>

      <Callout />

      <SectionHeader
        title="Batteries included"
        subtitle="Everything you need to build event-driven apps including:"
        counter="/05"
      />

      <div className="grid">
        <div className="grid-center-6 four-cols batteries sm-col-8-center">
          <div>
            <div className="icon">
              <CLI />
            </div>
            <p>Developer CLI</p>
          </div>

          <div>
            <div className="icon">
              <Retries />
            </div>
            <p>Automatic Retries</p>
          </div>

          <div>
            <div className="icon">
              <Replays />
            </div>
            <p>Manual Replays</p>
          </div>

          <div>
            <div className="icon">
              <Transforms />
            </div>
            <p>Payload Transforms</p>
          </div>

          <div>
            <div className="icon">
              <Versions />
            </div>
            <p>Version History</p>
          </div>

          <div>
            <div className="icon">
              <Rollback />
            </div>
            <p>Instant Rollbacks</p>
          </div>

          <div>
            <div className="icon">
              <Audit />
            </div>
            <p>Audit Trails</p>
          </div>

          <div>
            <div className="icon">
              <Logging />
            </div>
            <p>Full Logging</p>
          </div>

          <div>
            <div className="icon">
              <Historical />
            </div>
            <p>Historical Testing</p>
          </div>

          <div>
            <div className="icon">
              <Observability />
            </div>
            <p>End-to-end Observability</p>
          </div>

          <div>
            <div className="icon">
              <Alerting />
            </div>
            <p>Alerting</p>
          </div>

          <div>
            <div className="icon">
              <CLI />
            </div>
            <p>Developer CLI</p>
          </div>
        </div>
        <div className="grid-line" />
      </div>

      <Callout small="Still reading?" />

      <Footer />
      {/* Roboto Condensed is a weird font - way smaller than others -
      so we adjust the root size for this page only :/ */}
      <style>{`:root { font-size: 22px; }`} </style>
    </Wrapper>
  );
}

// Wrapper defines a top-level scope for nesting home-specific CSS classes within.
const Wrapper = styled.div`
  .code {
    padding: 2rem 0 10vh;
  }

  .sdk-list {
    display: flex;
    justify-content: center;
    margin: 2rem 0 0;
    .sdk-list-item {
      margin-left: 0.8em;
    }
  }

  /* Apply spacing prior to each header */
  .section-header > div {
    padding-top: var(--section-padding);
  }

  /* Automatically apply spacing to the section's content after the header */
  .section-header + .grid > div:first-of-type {
    padding-top: var(--header-trailing-padding);
  }

  .hero-grid {
    padding-top: var(--nav-height);
    margin-top: calc(var(--nav-height) * -1);
    background: url(/assets/hero-grid.svg) no-repeat right 10%;
    align-items: center;
    p {
      color: #fff;
    }
  }

  .img-mesh {
    border: 1px solid #000;
    border-radius: var(--border-radius);
  }

  .integrations {
    padding: 2rem 0;
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-gap: var(--grid-gap);
    p {
      color: #fff;
    }
  }

  .dx {
    align-items: center;
    grid-gap: var(--header-trailing-padding) var(--grid-gap);
    padding-bottom: var(--section-padding);

    svg {
      margin: 0 0 0.85rem;
    }

    h3 {
      margin: 0 0 0.75rem;
    }

    img {
      border: 1px solid rgba(var(--black-rgb), 0.5);
      box-shadow: 0 10px 5rem rgba(var(--black-rgb), 0.5);
      pointer-events: none;
      width: 100%;
      border-radius: var(--border-radius);
    }
  }

  .batteries {
    padding-bottom: var(--section-padding);

    > div {
      background: var(--black);
      border-radius: var(--border-radius);
      padding: 1rem;
      font-size: 1.2rem;
    }
    p {
      color: #fff;
    }
    .icon {
      height: 4rem;
      width: 4rem;
      margin: 0 0 1rem;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: var(--border-radius);
      background: var(--primary-color);
    }
  }

  @media (max-width: 800px) {
    .integrations {
      grid-template-columns: 1fr;
    }

    .dx > div {
      margin-top: 1rem;
    }
    .dx img {
      margin-bottom: 2rem;
    }
  }
`;

const Hero = styled.div`
  padding: 8em 0;

  display: grid;
  grid-template-columns: repeat(8, 1fr);
  grid-gap: var(--grid-gap);

  > div:first-of-type {
    grid-column: span 5;
  }

  img {
    grid-column: span 3;
    width: 100%;
  }

  p {
    padding: 0 0 1.5rem;
    font-family: var(--font);
  }

  .button {
    font-family: var(--font-mono);
    display: inline-block;
    letter-spacing: -0.5px;
  }
  .button-text-light {
    font-weight: 200;
  }
  .button-text-med {
    font-weight: 600;
  }

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
    padding: 8vh 0;

    > div:first-of-type {
      grid-column: 1;
    }

    img {
      display: none;
    }

    p {
      padding: 0 0 2rem;
    }

    .button {
      display: flex;
      align-self: stretch;
      margin: 0.5rem 0 0 0;
    }
  }
`;

const HIW = styled.div`
  > div {
    padding: var(--header-trailing-padding) 1rem 3rem 0;
  }
  svg {
    margin: 0 0 1rem;
  }

  @media (max-width: 800px) {
    > div {
      padding: 2vh 0 0 0;
    }
  }

  .grid-line {
    grid-row-start: 1;
    grid-row-end: 4;
  }
`;
