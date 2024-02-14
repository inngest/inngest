import Link from "next/link";
import clsx from "clsx";

import Logo from "src/shared/Icons/Logo";
import Discord from "../Icons/Discord";
import Github from "../Icons/Github";
import XSocialIcon from "../Icons/XSocialIcon";
import Container from "../layout/Container";
import footerLinks from "./footerLinks";
import StatusWidget from "../StatusWidget";
import FooterCallout from "./FooterCallout";

export default function Footer({
  ctaRef,
  disableCta = false,
}: {
  ctaRef?: string;
  disableCta?: boolean;
}) {
  return (
    <>
      {!disableCta && <FooterCallout ctaRef={ctaRef} />}
      <footer
        className={clsx(
          "border-t border-slate-800 bg-slate-1000",
          disableCta ? "mt-36" : "mt-12"
        )}
      >
        <Container className="pb-12 pt-12 lg:pt-24">
          <div className="xl:flex xl:gap-12 w-full rounded-lg relative ">
            <div className="mb-12 flex gap-6 items-start">
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
                      <XSocialIcon />
                      X.com
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
    </>
  );
}
