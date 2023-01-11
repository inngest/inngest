import Logo from "src/shared/Icons/Logo";
import Discord from "../Icons/Discord";
import Github from "../Icons/Github";
import Twitter from "../Icons/Twitter";
import Container from "../layout/Container";
import footerLinks from "./footerLinks";
import StatusWidget from "../StatusWidget";
import { Button } from "../Button";

export default function Footer() {
  return (
    <footer className="mt-20  bg-slate-1000">
      <div className="bg-indigo-600 backdrop-blur text-white">
        {/* <div className="bg-slate-950/40 backdrop-blur text-white"> */}
        <Container className="py-20 lg:py-32">
          <h4 className=" text-2xl lg:text-4xl tracking-tight mb-4 font-medium">
            Ready to start building?
          </h4>
          <p className="text-sm lg:text-base max-w-xl text-indigo-100">
            Enqueue and schedule reliable serverless functions in record time.
          </p>
          <code className="text-sm text-white block mt-8 mr-4">
            <span className="text-indigo-300">$</span> npx inngest-cli dev
          </code>
          <Button
            className="mt-4"
            arrow="right"
            variant="secondary"
            href="/sign-up?ref=callout"
          >
            Get started for free
          </Button>
        </Container>
      </div>
      <Container className="pb-12 pt-16 lg:pt-24">
        <div className="w-full rounded-lg relative ">
          <div className=" mb-12 flex gap-6 items-center">
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
                    href="https://discord.gg/EuesV2ZSnX"
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
