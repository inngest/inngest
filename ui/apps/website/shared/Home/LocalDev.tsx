import Link from 'next/link';

import Container from '../layout/Container';
import CopyBtn from './CopyBtn';
import Heading from './Heading';

export default function LocalDev({ className }: { className?: string }) {
  const handleCopyClick = (copy) => {
    navigator.clipboard?.writeText(copy);
  };
  return (
    <div
      style={{
        backgroundImage: 'url(/assets/pricing/table-bg.png)',
        backgroundPosition: 'center -50%',
        backgroundRepeat: 'no-repeat',
        backgroundSize: '2000px 1100px',
      }}
    >
      <Container className={`relative z-30 mt-44 ${className}`}>
        <div>
          <Heading
            title="Unparalleled Local Dev"
            lede="Our open source Inngest dev server runs on your machine for a complete
          local development experience, with production parity. Get instant feedback on
          your work and deploy to prod with full confidence."
            className="mx-auto max-w-3xl text-center"
          />

          <div className="mb-20 mt-12 flex flex-col items-center justify-center gap-4 md:flex-row">
            <div className="flex rounded bg-white/10 text-sm text-slate-200 shadow-lg backdrop-blur-md">
              <pre className=" py-2 pl-4 pr-2">
                <code className="bg-transparent text-slate-300">
                  <span>npx</span> inngest-cli dev
                </code>
              </pre>
              <div className="flex items-center justify-center rounded-r pl-2 pr-2.5">
                <CopyBtn btnAction={handleCopyClick} copy="npx inngest-cli@latest dev" />
              </div>
            </div>
            <Link
              href="/docs/quick-start?ref=homepage-dev-tools"
              className="whitespace-nowrap rounded-md border border-slate-800 bg-slate-800 px-3 py-1.5 text-sm text-white transition-all hover:border-slate-600 hover:bg-slate-500/10 hover:bg-slate-600"
            >
              Read the quick start guide
            </Link>
          </div>
        </div>
        <img
          src="/assets/homepage/dev-server-screenshot.jpg"
          alt="Inngest Dev Server Screenshot"
          className={`
          scale-80 pointer-events-none
          m-auto mt-14 w-full max-w-6xl origin-center
          rounded-lg
          border
          border-white/10 shadow-none
        `}
        />
      </Container>
    </div>
  );
}
