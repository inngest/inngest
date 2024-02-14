import Link from 'next/link';
import clsx from 'clsx';
import Logo from 'src/shared/Icons/Logo';

import Discord from '../Icons/Discord';
import Github from '../Icons/Github';
import XSocialIcon from '../Icons/XSocialIcon';
import StatusWidget from '../StatusWidget';
import Container from '../layout/Container';
import FooterCallout from './FooterCallout';
import footerLinks from './footerLinks';

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
        className={clsx('bg-slate-1000 border-t border-slate-800', disableCta ? 'mt-36' : 'mt-12')}
      >
        <Container className="pb-12 pt-12 lg:pt-24">
          <div className="relative w-full rounded-lg xl:flex xl:gap-12 ">
            <div className="mb-12 flex items-start gap-6">
              <Logo className="relative top-[3px] w-20 text-white" />
              <StatusWidget />
            </div>
            <div className="flex flex-wrap gap-8 lg:gap-12 xl:gap-20">
              {footerLinks.map((footerLink, i) => (
                <div className=" flex-shrink-0  lg:w-auto" key={i}>
                  <h4 className="mb-6 text-xs font-semibold uppercase text-slate-400">
                    {footerLink.name}
                  </h4>
                  <ul className="flex flex-col gap-4">
                    {footerLink.links.map((link, j) => (
                      <li key={j}>
                        <a
                          className="group flex items-center gap-1.5 text-sm text-white transition-all hover:text-indigo-400"
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
                <h4 className="mb-6 text-xs font-semibold uppercase text-slate-400">Community</h4>
                <ul className="flex flex-col gap-4">
                  <li>
                    <a
                      className="group flex items-center gap-2 text-sm text-white transition-all hover:text-indigo-400"
                      href="https://www.inngest.com/discord"
                    >
                      <Discord />
                      Discord
                    </a>
                  </li>
                  <li>
                    <a
                      className="group flex items-center gap-2 text-sm text-white transition-all hover:text-indigo-400"
                      href="https://github.com/inngest/inngest-js"
                    >
                      <Github />
                      GitHub
                    </a>
                  </li>
                  <li>
                    <a
                      className="group flex items-center gap-2 text-sm text-white transition-all hover:text-indigo-400"
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
          <ul className="mt-12 flex flex-col-reverse items-start lg:flex-row lg:gap-6">
            <li className=" py-1.5 text-center text-sm font-medium text-slate-300">
              &copy;
              {new Date().getFullYear()} Inngest Inc.
            </li>
            <li className=" text-center text-sm">
              <a
                className="block py-1.5 text-slate-400 transition-colors hover:text-indigo-400"
                href="/privacy?ref=footer"
              >
                Privacy
              </a>
            </li>
            <li className=" text-center text-sm">
              <a
                className="block py-1.5 text-slate-400 transition-colors hover:text-indigo-400"
                href="/terms?ref=footer"
              >
                Terms and Conditions
              </a>
            </li>
            <li className=" text-center text-sm">
              <a
                className="block py-1.5 text-slate-400 transition-colors hover:text-indigo-400"
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
