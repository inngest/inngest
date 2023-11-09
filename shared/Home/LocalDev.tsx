import Link from "next/link";
import Container from "../layout/Container";
import CopyBtn from "./CopyBtn";

import Heading from "./Heading";

export default function LocalDev({ className }: { className?: string }) {
  const handleCopyClick = (copy) => {
    navigator.clipboard?.writeText(copy);
  };
  return (
    <div
      style={{
        backgroundImage: "url(/assets/pricing/table-bg.png)",
        backgroundPosition: "center -50%",
        backgroundRepeat: "no-repeat",
        backgroundSize: "2000px 1100px",
      }}
    >
      <Container className={`mt-44 relative z-30 ${className}`}>
        <div>
          <Heading
            title="Unparalleled Local Dev"
            lede="Our open source Inngest dev server runs on your machine for a complete
          local development experience, with production parity. Get instant feedback on
          your work and deploy to prod with full confidence."
            className="mx-auto max-w-3xl text-center"
          />

          <div className="mt-12 mb-20 flex gap-4 flex-col md:flex-row items-center justify-center">
            <div className="bg-white/10 backdrop-blur-md flex rounded text-sm text-slate-200 shadow-lg">
              <pre className=" pl-4 pr-2 py-2">
                <code className="bg-transparent text-slate-300">
                  <span>npx</span> inngest-cli dev
                </code>
              </pre>
              <div className="rounded-r flex items-center justify-center pl-2 pr-2.5">
                <CopyBtn
                  btnAction={handleCopyClick}
                  copy="npx inngest-cli@latest dev"
                />
              </div>
            </div>
            <Link
              href="/docs/quick-start?ref=homepage-dev-tools"
              className="rounded-md px-3 py-1.5 text-sm bg-transparent transition-all text-white border border-slate-800 hover:border-slate-600 hover:bg-slate-500/10 whitespace-nowrap"
            >
              Get Started
            </Link>
          </div>
        </div>
        <img
          src="/assets/homepage/dev-server-screenshot.jpg"
          alt="Inngest Dev Server Screenshot"
          className={`
          mt-14 w-full
          rounded-lg shadow-none m-auto scale-80 origin-center
          pointer-events-none
          max-w-6xl
          border border-white/10
        `}
        />
      </Container>
    </div>
  );
}
