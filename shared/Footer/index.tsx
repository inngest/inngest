import Link from "next/link";

import Logo from "src/shared/Icons/Logo";
import Discord from "../Icons/Discord";
import Github from "../Icons/Github";
import Twitter from "../Icons/Twitter";
import Container from "../layout/Container";
import footerLinks from "./footerLinks";
import StatusWidget from "../StatusWidget";

const MeshGradient = `
radial-gradient(at 35% 95%, hsla(258,82%,61%,1) 0px, transparent 50%),
radial-gradient(at 65% 9%, hsla(261,49%,53%,1) 0px, transparent 50%),
radial-gradient(at 77% 89%, hsla(246,66%,61%,1) 0px, transparent 50%),
url(/assets/textures/wave-large.svg)
`;

export default function Footer({ ctaRef }: { ctaRef?: string }) {
  return (
    <footer className="mt-80 border-t border-slate-800 bg-slate-1000">
      <Container>
        <div className="p-2.5 rounded-[14px] bg-slate-800/50 -mt-60">
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
              Ready to start building?
            </h4>
            <p className="mt-4 mb-6 text-sm lg:text-base text-indigo-100 drop-shadow">
              Ship background functions & workflows like never before
            </p>
            <div className="flex flex-col gap-5 items-center">
              <code className="mt-8 py-2.5 px-5 rounded-[6px] bg-white/10 text-sm text-white backdrop-blur-md font-medium">
                <span className="">$</span> npx inngest-cli dev
              </code>
              <Link
                href={`/sign-up?ref=${
                  ctaRef ? `${ctaRef}-callout` : "callout"
                }`}
                className="py-3 px-5 bg-slate-800 text-white rounded-[6px] text-sm transition-all hover:bg-slate-900"
              >
                Get started for free
              </Link>
            </div>
          </div>
        </div>
      </Container>

      <Container className="pb-12 pt-16 lg:pt-24">
        <div className="xl:flex xl:gap-12 w-full rounded-lg relative ">
          <div className=" mb-12 flex gap-6 items-start">
            <Logo className="text-white w-20 relative top-[3px]" />
            <StatusWidget />
          </div>
          <div className="flex flex-wrap gap-8 lg:gap-12 xl:gap-20">
            {footerLinks.map((footerLink, i) => (
              <div className=" lg:w-auto  flex-shrink-0" key={i}>
                <h4 className="text-slate-400 text-xs uppercase font-semibold mb-6">
                  {footerLink.name}
                </h4>
                <ul className="flex flex-col gap-4">
                  {footerLink.links.map((link, j) => (
                    <li key={j}>
                      <a
                        className="text-white text-sm flex items-center group gap-1.5 hover:text-indigo-400 transition-all"
                        href={link.url}
                      >
                        {link.icon && <link.icon size={22} color="indigo" />}
                        {link.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            ))}

            <div>
              <h4 className="text-slate-400 text-xs uppercase font-semibold mb-6">
                Community
              </h4>
              <ul className="flex flex-col gap-4">
                <li>
                  <a
                    className="text-white text-sm flex items-center group gap-2 hover:text-indigo-400 transition-all"
                    href="https://www.inngest.com/discord"
                  >
                    <Discord />
                    Discord
                  </a>
                </li>
                <li>
                  <a
                    className="text-white text-sm flex items-center group gap-2 hover:text-indigo-400 transition-all"
                    href="https://github.com/inngest/inngest-js"
                  >
                    <Github />
                    GitHub
                  </a>
                </li>
                <li>
                  <a
                    className="text-white text-sm flex items-center group gap-2 hover:text-indigo-400 transition-all"
                    href="https://twitter.com/inngest"
                  >
                    <Twitter />
                    Twitter
                  </a>
                </li>
              </ul>
            </div>
          </div>
        </div>
        <ul className="flex mt-12 lg:gap-6 flex-col-reverse items-start lg:flex-row">
          <li className=" text-sm text-center py-1.5 text-slate-300 font-medium">
            &copy;
            {new Date().getFullYear()} Inngest Inc.
          </li>
          <li className=" text-sm text-center">
            <a
              className="text-slate-400 py-1.5 block hover:text-indigo-400 transition-colors"
              href="/privacy?ref=footer"
            >
              Privacy
            </a>
          </li>
          <li className=" text-sm text-center">
            <a
              className="text-slate-400 py-1.5 block hover:text-indigo-400 transition-colors"
              href="/terms?ref=footer"
            >
              Terms and Conditions
            </a>
          </li>
          <li className=" text-sm text-center">
            <a
              className="text-slate-400 py-1.5 block hover:text-indigo-400 transition-colors"
              href="/security?ref=footer"
            >
              Security
            </a>
          </li>
        </ul>
      </Container>
    </footer>
  );
}
