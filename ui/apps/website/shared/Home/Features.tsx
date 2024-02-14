import Link from 'next/link';
import { ArrowPathRoundedSquareIcon } from '@heroicons/react/20/solid';
import {
  ArrowDownOnSquareStackIcon,
  BoltSlashIcon,
  ChevronDoubleRightIcon,
  ExclamationCircleIcon,
  MoonIcon,
  PlayPauseIcon,
  Square3Stack3DIcon,
} from '@heroicons/react/24/outline';

import Replay from '../Icons/Replay';
import Container from '../layout/Container';
import Heading from './Heading';

const Code = ({ children }) => (
  <code className="bg-transparent px-0 font-mono text-indigo-200">{children}</code>
);

const content = [
  {
    title: 'Automatic retries',
    Icon: ArrowPathRoundedSquareIcon,
    content: (
      <p>
        Every step of your function is retried whenever it throws an error. Customize the number of
        retries to ensure your functions are reliably executed.
      </p>
    ),
    ctas: [
      {
        href: '/docs/functions/retries',
        text: 'Learn about retries',
      },
    ],
  },
  {
    title: 'Durable sleep',
    Icon: MoonIcon,
    content: (
      <p>
        Pause your function for hours, days or weeks with <Code>step.sleep()</Code> and{' '}
        <Code>step.sleepUntil()</Code>. Inngest stores the state of your functions and resumes
        execution automatically exactly when it should.
      </p>
    ),
    ctas: [
      {
        href: '/docs/reference/functions/step-sleep',
        text: 'Learn about sleep',
      },
      {
        href: '/docs/reference/functions/step-sleep-until',
        text: 'Learn about sleepUntil',
      },
    ],
  },
  {
    title: 'Manage concurrency',
    Icon: ChevronDoubleRightIcon,
    content: (
      <p>
        Set custom concurrency limits for every function to fine-tune how quickly your jobs run. For
        more control, set a <Code>key</Code> to create infinite "sub-queues" to control concurrency
        at any level.
      </p>
    ),
    ctas: [
      {
        href: '/docs/functions/concurrency',
        text: 'Learn about concurrency',
      },
    ],
  },
  {
    title: 'Rate limit + debounce',
    Icon: ArrowDownOnSquareStackIcon,
    content: (
      <p>
        Control how your functions are executed in a given time period. You can also use a custom
        key to set per-user or per-whatever rate limits or debounces with a single line of code.
      </p>
    ),
    ctas: [
      {
        href: '/docs/reference/functions/rate-limit',
        text: 'Learn about rate limit',
      },
      {
        href: '/docs/reference/functions/debounce',
        text: 'Learn about debounce',
      },
    ],
  },
  {
    title: 'Declarative job cancellation',
    Icon: BoltSlashIcon,
    content: (
      <p>
        Cancel jobs just by sending an event. No need to keep track of running jobs, Inngest can
        automatically match long running functions with cancellation events to kill jobs
        declaratively.
      </p>
    ),
    ctas: [
      {
        href: '/docs/functions/cancellation',
        text: 'Learn about cancellation',
      },
    ],
  },
  {
    title: 'Custom failure handlers',
    Icon: ExclamationCircleIcon,
    content: (
      <p>
        Define failure handlers along side your function code and Inngest will automatically run
        them when things go wrong. Use it to handle rollback, send an email or trigger an alert for
        your team.
      </p>
    ),
    ctas: [
      {
        href: '/docs/reference/functions/handling-failures',
        text: 'Learn about handling failures',
      },
    ],
  },
  {
    title: 'Pause functions for additional input',
    Icon: PlayPauseIcon,
    content: (
      <p>
        Use <Code>step.waitForEvent()</Code> to pause your function until another event is received.
        Create human-in the middle workflows or communicate between long running jobs with events.
      </p>
    ),
    ctas: [
      {
        href: '/docs/reference/functions/step-wait-for-event',
        text: 'Learn about waiting for events',
      },
    ],
  },
  {
    title: 'Batching for high load',
    Icon: Square3Stack3DIcon,
    content: (
      <p>
        Reduce the load on your system and save money by automatically batching bursty data or high
        volume.
      </p>
    ),
    ctas: [
      {
        href: '/docs/guides/batching',
        text: 'Learn about batching',
      },
    ],
  },

  {
    title: 'Replay functions',
    Icon: Replay,
    content: (
      <p>
        Forget dead letter queues. Fix your issues then replay a failed function in a single click.
      </p>
    ),
    ctas: [
      {
        href: '/docs/platform/replay',
        text: 'Learn about replay',
      },
    ],
  },
];

export default function Features() {
  return (
    <Container className="mt-24">
      <Heading
        title="We built it, so you don't have to"
        lede={
          <>
            Building reliable backends is hard. Don't waste weeks building out bespoke systems:{' '}
            <br />
            We've built in all the tools that you need to create complex backend workflows.
          </>
        }
        className="text-center"
      />
      <div className="mx-auto my-24">
        <div className="mx-auto grid gap-x-8 gap-y-12 md:grid-cols-2 lg:grid-cols-3 lg:gap-x-12 lg:gap-y-20">
          {content.map(({ title, Icon = ArrowPathRoundedSquareIcon, content, ctas }) => (
            <div className="flex flex-col gap-6 text-sm font-medium leading-normal tracking-normal text-slate-300">
              <div className="flex items-center gap-3 text-slate-400/80">
                <Icon className="w-6" />
                <h3 className="text-lg font-semibold text-slate-50">{title}</h3>
              </div>
              <div className="flex-grow">{content}</div>
              <div className="flex gap-4">
                {ctas?.length > 0 &&
                  ctas.map(({ href, text = 'Learn more' }) => (
                    <Link
                      href={href}
                      className="text-indigo-300 decoration-slate-50/30 decoration-dotted underline-offset-4 hover:text-white hover:underline"
                    >
                      {text} →
                    </Link>
                  ))}
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className="flex items-center">
        <Link
          href="/docs"
          className="mx-auto whitespace-nowrap rounded-md border border-slate-800 bg-slate-800 px-6 py-2 font-medium text-white transition-all hover:border-slate-600 hover:bg-slate-500/10 hover:bg-slate-600"
        >
          Explore our documentation and guides
        </Link>
      </div>
    </Container>
  );
}
