import Link from "next/link";

import Container from "../layout/Container";

const MeshGradient = `
radial-gradient(at 35% 95%, hsla(258,82%,61%,1) 0px, transparent 50%),
radial-gradient(at 65% 9%, hsla(261,49%,53%,1) 0px, transparent 50%),
radial-gradient(at 77% 89%, hsla(246,66%,61%,1) 0px, transparent 50%),
url(/assets/textures/wave-large.svg)
`;

export default function FooterCallout({
  title = "Ready to start building?",
  description = "Ship background functions & workflows like never before",
  ctaHref = process.env.NEXT_PUBLIC_SIGNUP_URL,
  ctaText = "Get started for free",
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
      <div className="p-2.5 rounded-[14px] bg-slate-800/50 mt-28 mb-12">
        <div
          className="py-12 lg:py-16 text-center rounded-lg shadow"
          style={{
            backgroundColor: `hsla(222,79%,61%,1)`,
            backgroundImage: MeshGradient,
            backgroundSize: "cover",
            backgroundPosition: "100%",
          }}
        >
          <h4 className="text-2xl lg:text-3xl tracking-tight mb-4 font-semibold drop-shadow">
            {title}
          </h4>
          <p className="mt-4 mb-6 text-sm lg:text-base text-indigo-100 drop-shadow">
            {description}
          </p>
          <div className="flex flex-col gap-5 items-center">
            {showCliCmd && (
              <code className="mt-8 py-2.5 px-5 rounded-[6px] bg-white/10 text-sm text-white backdrop-blur-md font-medium">
                <span className="">$</span> npx inngest-cli dev
              </code>
            )}
            <Link
              href={`${ctaHref}?ref=${
                ctaRef ? `${ctaRef}-callout` : "callout"
              }`}
              className="py-3 px-5 bg-slate-800 text-white rounded-[6px] text-sm transition-all hover:bg-slate-900"
            >
              {ctaText}
            </Link>
          </div>
        </div>
      </div>
    </Container>
  );
}
