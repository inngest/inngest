import Footer from "../../shared/footer";
import Nav from "../../shared/nav";
import Content from "../../shared/content";
import { Wrapper, Inner } from "../../shared/blog";

export default function BlogLayout() {
  return (
    <>
      <Wrapper>
        <Nav />
        <Content>
          <Inner>
            <h1 id="introducing-inngest--an-event-workflow-platform">
              Introducing Inngest: an event workflow platform
            </h1>
            <div className="blog--date">October 2021</div>
            <div className="blog--callout">
              We’re launching Inngest, a platform designed to make building
              event-driven systems fast and easy.
              <p>
                First, what is Inngest? Inngest is a serverless event platform.
                It aggregates events from your internal and external systems,
                then allows you automatically run serverless code in response to
                these events - as a multi-step workflow.{" "}
              </p>
              <p>
                It’s like putting GitHub Actions, Lambda, Segment, and Zapier in
                a blender. You can sign up to Inngest for free and start today
                by <a href="https://app.inngest.com/register">visiting here</a>.
              </p>
            </div>

            <h2 id="why-events">Why events?</h2>
            <p>
              Events drive the world. They describe exactly what happens in all
              of your systems. When a user signs up, that’s an event. When a
              user pays - or fails to pay - that’s an event. When you update a
              task, or a Salesforce lead - that’s an event.
            </p>
            <p>
              Events represent everything, as it happens in real time. For the
              engineers, it’s also decoupled from the implementation - you can
              change how signup works, but the event is still the same (someone
              has still signed up, after all).
            </p>
            <p>Unifying and working with these events makes a lot possible.</p>
            <h2 id="what-can-it-be-used-for--some-examples">
              What can it be used for? Some examples…
            </h2>
            <p>
              High level, when you begin working with events you can react to
              anything - in real time. That means you can do things like:
            </p>
            <ul>
              <li>Build real-time sync between platforms as things update</li>
              <li>
                Run workflows when users sign up, handling the emails,
                marketing, billing organization, and internal flows
                asynchronously
              </li>
              <li>
                Process leads, enrich data, and assign tasks when your company
                gets new leads
              </li>
            </ul>
            <p>
              And when you unify your events, you can coordinate between
              multiple events (or their absence):
            </p>
            <ul>
              <li>
                When a user signs up but you don’t receive a sign-in event
                within 7 days, run a churn campaign and send emails to the user.
              </li>
              <li>
                When a shipping label is generated but no shipment sent event is
                received by the label expiry, generate a new label and email the
                user.
              </li>
            </ul>
            <h2 id="some-extra-benefits">Some extra benefits</h2>
            <p>
              By building with events as a first class citizen you also get:
            </p>
            <ul>
              <li>Reproducibility</li>
              <li>Traceability</li>
              <li>Easy debugging</li>
              <li>Retries</li>
              <li>Audit trails</li>
              <li>Versioning and change management</li>
              <li>Event coordination</li>
            </ul>
            <p>
              The plumbing exists for this in common platforms. You can use SQS,
              Kafka, Lambda, Terraform. etc. to start building event-driven
              systems yourselves. Though, it can take some time to build the
              debuggability, audit trails, and versioning you get out of the box
              with Inngest.
            </p>
            <p>
              We’ll dive into more of these pieces in future posts. If you’re
              interested in getting started with a serverless event platform in
              minutes, check out{" "}
              <a href="https://www.inngest.com">https://www.inngest.com</a>.
            </p>
          </Inner>
        </Content>
        <Footer />
      </Wrapper>
    </>
  );
}
