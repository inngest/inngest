import Logo from "src/shared/Icons/Logo";
import Discord from "../Icons/Discord";
import Github from "../Icons/Github";
import Twitter from "../Icons/Twitter";
import Container from "../layout/Container";
import footerLinks from "./footerLinks";
import StatusWidget from "../StatusWidget";

export default function Footer() {
  return (
    <footer
      className="mt-20  bg-slate-1000"
      style={{
        backgroundImage: "url(/assets/footer/footer-grid.svg)",
        backgroundSize: "cover",
        backgroundPosition: "right -40px top",
        backgroundRepeat: "no-repeat",
      }}
    >
      <Container>
        <div className="relative">
          <div className=" w-full py-16 rounded-lg relative ">
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
                          className="text-white text-sm flex group gap-1 hover:text-indigo-400 transition-all"
                          href={link.url}
                        >
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
                      className="text-white text-sm flex items-center group gap-1 hover:text-indigo-400 transition-all"
                      href="https://discord.gg/EuesV2ZSnX"
                    >
                      <Discord />
                      Discord
                    </a>
                  </li>
                  <li>
                    <a
                      className="text-white text-sm flex items-center group gap-1 hover:text-indigo-400 transition-all"
                      href="https://github.com/inngest/inngest-js"
                    >
                      <Github />
                      GitHub
                    </a>
                  </li>
                  <li>
                    <a
                      className="text-white text-sm flex items-center group gap-1 hover:text-indigo-400 transition-all"
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
        </div>
        <ul className="lg:py-0 flex lg:gap-6 flex-col-reverse items-start lg:flex-row">
          <li className=" text-sm text-center py-1.5 lg:pb-20 text-slate-300 font-medium">
            &copy;
            {new Date().getFullYear()} Inngest Inc.
          </li>
          <li className=" text-sm text-center">
            <a
              className="text-slate-400 py-1.5 lg:pb-20 block hover:text-indigo-400 transition-colors"
              href="/privacy?ref=footer"
            >
              Privacy
            </a>
          </li>
          <li className=" text-sm text-center">
            <a
              className="text-slate-400 py-1.5 lg:pb-20 block hover:text-indigo-400 transition-colors"
              href="/terms?ref=footer"
            >
              Terms and Conditions
            </a>
          </li>
          <li className=" text-sm text-center">
            <a
              className="text-slate-400 py-1.5 lg:pb-20 block hover:text-indigo-400 transition-colors"
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
