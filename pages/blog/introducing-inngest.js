import Footer from "../../shared/footer";
import Nav from "../../shared/nav";
import Content from "../../shared/content";
import { Wrapper, Inner } from "../../shared/blog";

export default function BlogLayout() {
  return (
    <Wrapper>
      <Nav />
      <Content>
        <Inner>
          <h1>Introducing Inngest: an event workflow platform</h1>
          <div className="blog--date">2021-10-05</div>
          <div className="blog--callout">
            <p>
              We’re launching Inngest, a platform designed to make building
              event-driven systems fast and easy.
            </p>
            <p>
              First, what is Inngest? Inngest is a serverless event platform. It
              aggregates events from your internal and external systems, then
              allows you automatically run serverless code in response to these
              events - as a multi-step workflow.{" "}
            </p>
            <p>
              It’s like putting GitHub Actions, Lambda, Segment, and Zapier in a
              blender. You can build server-side functions and glue code in
              minutes, with no servers. If you&#39;re interested, you can sign
              up to Inngest for free and start today by{" "}
              <a href="https://app.inngest.com/register">visiting here</a>.
            </p>
          </div>
          <h2 id="why-events">Why events?</h2>
          <p>
            Events are powerful: they describe exactly what happens in every
            system. When a user signs up, that’s an event. When a user pays
            (or... fails to pay), that’s an event. When you update a task, or a
            Salesforce lead, or a GitHub PR that’s an event.
          </p>
          <p>
            So,{" "}
            <strong>events represent things as they happen in real time</strong>
            . You&#39;re probably familiar with them already because, well,
            product analytics has been a thing for some time.
          </p>
          <p>
            But they&#39;re powerful not just because of analytics. They&#39;re
            powerful because your systems often need to run a bunch of logic
            when things happen. For example, when a user signs up to your
            account you might need to add them to your billing system, add them
            to marketing lists, add them to your sales tools, add them to your
            CRM, and send them an email.
          </p>
          <p>
            To start, you might chuck the first integration in your API
            controller. Or a goroutine. And as things progress you might start
            building out queues, or if you&#39;re all in on microservices you
            might want to build a lot of infrastructure around sending messages
            - events.
          </p>
          <p>
            The beautiful thing about events is that{" "}
            <strong>
              events are decoupled from the actual implementation that creates
              them
            </strong>
            . You can change how signup works (oauth, magic links, or - have
            mercy - saml), but the event is still the same.{" "}
            <strong>It gives you freedom</strong>.
          </p>
          <p>Unifying and working with these events makes a lot possible.</p>
          <h2 id="what-is-inngest">What is Inngest?</h2>
          <p>
            So events are great. They let you know what&#39;s happening. They
            provide audit trails when things happen.{" "}
            <strong>But event-driven systems can be difficult to build</strong>.
            And they&#39;re very difficult to audit and debug. Don&#39;t get us
            wrong: if you want to wrangle with Terraform, maybe set up Kafka (I
            have a soft spot for NATS), build your publishers, subscribers,
            service discovery, throttling, retries, backoff, and other stuff, it
            can be done. But it&#39;s not exactly &quot;move fast&quot;, even if
            it is very much &quot;break things&quot;. You also don&#39;t get
            webhook handling, integrating with external services, change
            management, or non-technical insight here for free either.
          </p>
          <p>
            Well, this is where we step in.{" "}
            <strong>
              Inngest provides you with a serverless event-driven platform and
              DAG-based serverless functions out of the box.
            </strong>{" "}
            Send us events - any and all of them from your own systems. Connect
            webhooks up to external systems. And then build out serverless
            functions that run in real-time whenever events are received.
            That&#39;s the short version.
          </p>
          <p>The long version is that you can:</p>
          <ul>
            <li>
              Coordinate between events easily (eg. &quot;wait up to 1h for a
              customer to respond&quot;), while also handling timeouts
            </li>
            <li>Version each workflow, and roll back instantly if need be</li>
            <li>Visually see and understand the workflows</li>
            <li>Get retries and error handling out of the box</li>
            <li>Step-debug thtourhg your workflows</li>
            <li>
              Run <em>any</em> code as part of a workflow, in any language
            </li>
            <li>Automatically create schemas for each of your events</li>
            <li>Collaborate and hand-off workflows to non-developer folk</li>
          </ul>
          <p>
            We’ll dive into more of these pieces in future posts. If you’re
            interested in getting started, you can{" "}
            <a href="https://app.inngest.com/register">sign up here</a>.
          </p>
          <h2 id="what-can-it-be-used-for--some-examples">
            What can it be used for? Some examples…
          </h2>
          <p>
            Let&#39;s make things concrete. There are a few examples that are
            extremely common:
          </p>
          <ul>
            <li>
              On signup, propagate the new account to external systems (sales,
              marketing, customer support, billing) and send emails
            </li>
            <li>
              On signup, wait for X events to happen or begin a churn workflow
              (drip campaigns, etc).
            </li>
            <li>On payment, send receipts and update external systems</li>
            <li>
              When new customer support requests are received, run escalation
              logic procedures based off of the user&#39;s account, run NLP to
              detect importance, tone, and category of request
            </li>
          </ul>
          <p>
            There are also a bunch of things you might need to do depending on
            your sector:
          </p>
          <ul>
            <li>
              When a return label is generated but no shipment sent event is
              received by the label expiry, generate a new label and email the
              user.
            </li>
            <li>When a meeting is upcoming, send reminders</li>
          </ul>
          <p>
            It goes on, and on, and on, depending on what you&#39;re doing. And
            we&#39;re here to help you build it. You can{" "}
            <a href="https://app.inngest.com/register">get started for free</a>,
            or if you&#39;re interested in chatting with us you can send us an
            email:{" "}
            <a href="mailto:founders@inngest.com">founders@inngest.com</a>
          </p>
        </Inner>
      </Content>
      <Footer />
    </Wrapper>
  );
}
