import Link from 'next/link';

import Container from '../layout/Container';
import Heading from './Heading';

const highlights = [
  {
    title: 'Serverless, Servers or Edge',
    description:
      'Inngest functions run anywhere that you deploy your code. Mix and match for your needs, from GPU optimized VMs to instantly scaling serverless platforms.',
    img: '/assets/homepage/paths-graphic.svg',
  },
  {
    title: 'Logging & observability built-in',
    description: 'Debug issues quickly without having to leave the Inngest dashboard.',
    img: '/assets/homepage/observability-graphic.svg',
  },
  {
    title: 'We call you',
    description:
      'Inngest invokes your code via HTTP at exactly the right time, injecting function state on each call.  Ship complex workflows by writing code.',
    img: '/assets/homepage/we-call-you-graphic.svg',
  },
];

export default function RunAnywhere() {
  return (
    <Container className="mb-24 mt-40 tracking-tight lg:mt-64">
      <Heading
        title={
          <>
            Run anywhere, zero infrastructure
            <br className="hidden lg:block" /> or config required
          </>
        }
        lede="Inngest calls your code wherever it's hosted. Deploy to your existing setup, and deliver products faster without managing infrastructure."
        className="mx-auto max-w-3xl text-center"
      />

      <div className="mx-auto mb-24 mt-8 grid max-w-6xl gap-7 md:grid-cols-3 lg:my-24">
        {highlights.map(({ title, description, img }, idx) => (
          <div
            key={idx}
            style={{
              backgroundImage: 'radial-gradient(114.31% 100% at 50% 0%, #131E38 0%, #0A1223 100%)',
            }}
            className="flex flex-col justify-between rounded-lg "
          >
            <div className="mx-9 my-6">
              <h3 className="text-xl font-semibold">{title}</h3>
              <p className="my-1.5 text-sm font-medium text-indigo-200">{description}</p>
            </div>
            <img src={img} className="pointer-events-none w-full" alt={`Graphic for ${title}`} />
          </div>
        ))}
      </div>
    </Container>
  );
}
