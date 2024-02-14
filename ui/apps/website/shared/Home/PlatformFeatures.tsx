import Link from 'next/link';
import clsx from 'clsx';

import Container from '../layout/Container';
import Heading from './Heading';

const highlights = [
  {
    title: 'Ship reliable code',
    description:
      'All functions are retried automatically. Manage concurrency, rate limiting and backoffs in code within your function.',
    img: '/assets/homepage/platform/ship-code.png',
  },
  {
    title: 'Powerful scheduling',
    description:
      'Enqueue future work, sleep for months, and dynamically cancel jobs without managing job state or hacking APIs together.',
    img: '/assets/homepage/platform/powerful-scheduling.png',
  },
  {
    title: 'Replay functions at any time',
    description:
      'Forget the dead letter queue. Replay functions that have failed, or replay functions in your local environment to debug issues easier than ever before.',
    img: '/assets/homepage/platform/replay-functions.png',
  },
];

export default function PlatformFeatures() {
  return (
    <Container className="my-44 tracking-tight">
      <Heading
        title="Giving developers piece of mind"
        lede="Inngest gives you everything you need with sensible defaults."
        className="mx-auto max-w-3xl text-center"
      />

      <div className="mx-auto my-24 flex max-w-6xl flex-col gap-8">
        {highlights.map(({ title, description, img }, idx) => (
          <div
            key={idx}
            className={clsx(
              `flex flex-col items-stretch rounded-xl border border-slate-900 bg-slate-950 p-2.5`,
              idx % 2 === 0 ? `lg:flex-row-reverse` : `lg:flex-row`
            )}
          >
            <div className=" flex flex-col justify-center px-6 py-6 lg:px-10 lg:py-12">
              <h3 className="text-xl font-semibold text-indigo-50">{title}</h3>
              <p className="my-1.5 text-sm text-indigo-200 lg:text-base">{description}</p>
            </div>
            <div className="bg-slate-1000 flex w-full items-center justify-center rounded">
              <img src={img} alt={`Graphic for ${title}`} className="m-auto w-full max-w-[600px]" />
            </div>
          </div>
        ))}
      </div>
    </Container>
  );
}
