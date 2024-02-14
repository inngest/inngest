import Link from 'next/link';

import Container from '../layout/Container';

const MeshGradient = `
radial-gradient(at 35% 95%, hsla(258,82%,61%,1) 0px, transparent 50%),
radial-gradient(at 65% 9%, hsla(261,49%,53%,1) 0px, transparent 50%),
radial-gradient(at 77% 89%, hsla(246,66%,61%,1) 0px, transparent 50%),
url(/assets/textures/wave-large.svg)
`;

export default function FooterCallout({
  title = 'Ready to start building?',
  description = 'Ship background functions & workflows like never before',
  ctaHref = process.env.NEXT_PUBLIC_SIGNUP_URL,
  ctaText = 'Get started for free',
  ctaRef,
  showCliCmd = true,
}: {
  title?: React.ReactNode;
  description?: React.ReactNode;
  ctaHref?: string;
  ctaText?: string;
  ctaRef?: string;
  showCliCmd?: boolean;
}) {
  return (
    <Container>
      <div className="mb-12 mt-28 rounded-[14px] bg-slate-800/50 p-2.5">
        <div
          className="rounded-lg py-12 text-center shadow lg:py-16"
          style={{
            backgroundColor: `hsla(222,79%,61%,1)`,
            backgroundImage: MeshGradient,
            backgroundSize: 'cover',
            backgroundPosition: '100%',
          }}
        >
          <h4 className="mb-4 text-2xl font-semibold tracking-tight drop-shadow lg:text-3xl">
            {title}
          </h4>
          <p className="mb-6 mt-4 text-sm text-indigo-100 drop-shadow lg:text-base">
            {description}
          </p>
          <div className="flex flex-col items-center gap-5">
            {showCliCmd && (
              <code className="mt-8 rounded-[6px] bg-white/10 px-5 py-2.5 text-sm font-medium text-white backdrop-blur-md">
                <span className="">$</span> npx inngest-cli dev
              </code>
            )}
            <Link
              href={`${ctaHref}?ref=${ctaRef ? `${ctaRef}-callout` : 'callout'}`}
              className="rounded-[6px] bg-slate-800 px-5 py-3 text-sm text-white transition-all hover:bg-slate-900"
            >
              {ctaText}
            </Link>
          </div>
        </div>
      </div>
    </Container>
  );
}
