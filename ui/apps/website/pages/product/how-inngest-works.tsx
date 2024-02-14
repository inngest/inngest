import React from 'react';
import Link from 'next/link';
import HowInngestWorksGraphic from 'src/shared/Graphics/HowInngestWorks';

import Footer from '../../shared/Footer';
import Header from '../../shared/Header';
import Container from '../../shared/layout/Container';

export async function getStaticProps() {
  return {
    props: {
      designVersion: '2',
      meta: {
        title: 'How Inngest Works',
        description:
          'Learn the basics of how Inngest helps you ship background jobs and serverless queues faster than ever.',
      },
    },
  };
}

export default function HowInngestWorks() {
  return (
    <div>
      <Header />

      <div
        className="pb-12"
        style={{
          backgroundImage: 'url(/assets/pricing/table-bg.png)',
          backgroundPosition: 'center -30px',
          backgroundRepeat: 'no-repeat',
          backgroundSize: '1800px 1200px',
        }}
      >
        <Container>
          <h1 className="mt-12 text-3xl font-semibold tracking-tight text-white md:mt-20 lg:text-5xl">
            How Inngest Works
          </h1>
          <p className="text-xl text-slate-100">Learn the basics of Inngest</p>

          <div className="my-12 max-w-2xl">
            <p>
              Using the <a href="/docs/quick-start?ref=product-how-it-works">Inngest SDK</a>, you
              define your functions within your existing codebase and deploy your application
              anywhere. You define what events should trigger which functions and Inngest handles
              the rest.
            </p>
          </div>

          <section className="flex flex-col gap-8">
            <div className="flex items-center justify-center">
              <HowInngestWorksGraphic />
            </div>
            <div className="mx-auto mb-12 flex max-w-2xl flex-col gap-8 text-lg font-medium">
              <p>The lifecycle of a background job starts in your application with an event.</p>
              <ol className="flex list-decimal flex-col gap-4">
                <li>
                  Your application uses the Inngest SDK to{' '}
                  <InlineHighlight>send events within your code</InlineHighlight>, for example, at
                  the end of a user signup flow.
                </li>
                <li>
                  Inngest receives the event and immediately{' '}
                  <InlineHighlight>stores the full event history</InlineHighlight> for logging and
                  future retries and replays.
                </li>
                <li>
                  Inngest then determines if this event should trigger one or more of your functions
                  and then <InlineHighlight>schedules and enqueues jobs</InlineHighlight>.
                </li>
                <li>
                  Inngest then reads from the queue and{' '}
                  <InlineHighlight>executes jobs</InlineHighlight> via HTTP and{' '}
                  <InlineHighlight>manages state</InlineHighlight> across retries and multiple
                  steps.
                </li>
                <li>
                  Your application receives the request via the SDK's <a href="">serve</a> handler
                  and the <InlineHighlight>job runs in your application</InlineHighlight>.
                </li>
              </ol>
            </div>
          </section>
        </Container>
      </div>
      <Footer />
    </div>
  );
}

function InlineHighlight({ children }: { children: React.ReactNode }) {
  return (
    <span className="italic text-indigo-300 underline decoration-indigo-200/50 decoration-2	underline-offset-4">
      {children}
    </span>
  );
}
