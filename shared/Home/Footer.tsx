import Logo from "src/shared/Icons/Logo";
import Discord from "../Icons/Discord";
import Github from "../Icons/Github";
import Twitter from "../Icons/Twitter";

export default function Footer() {
  const footerLinks = [
    {
      name: "Product",
      links: [
        {
          label: "Function SDK",
          url: "/features/sdk?ref=footer",
        },
        {
          label: "Step Functions",
          url: "/features/step-functions?ref=footer",
        },
        {
          label: "Documentation",
          url: "/docs?ref=footer",
        },
        {
          label: "Patterns: Async + Event-Driven",
          url: "/patterns?ref=footer",
        },
        {
          label: "Self Hosting",
          url: "/docs/self-hosting?ref=footer",
        },
      ],
    },
    {
      name: "Use Cases",
      links: [
        {
          label: "Scheduled & cron jobs",
          url: "/uses/serverless-cron-jobs?ref=footer",
        },
        {
          label: "Node.js background jobs",
          url: "/uses/serverless-node-background-jobs?ref=footer",
        },
        {
          label: "Internal tools",
          url: "/uses/internal-tools?ref=footer",
        },
        {
          label: "User Journey Automation",
          url: "/uses/user-journey-automation?ref=footer",
        },
      ],
    },
    {
      name: "Company",
      links: [
        {
          label: "About",
          url: "/about?ref=footer",
        },
        {
          label: "Blog",
          url: "/blog?ref=footer",
        },
        {
          label: "Contact Us",
          url: "/contact?ref=footer",
        },
      ],
    },
  ];
  return (
    <footer className="mt-20 ">
      <div className="relative max-w-[1800px] m-auto px-10 z-10">
        <div className="absolute inset-0 rounded-lg bg-slate-900 opacity-20 rotate-1 -z-0 scale-[102%] mx-5"></div>
        <div
          className="px-8 lg:px-16 xl:px-20 w-full bg-slate-950 py-16 rounded-lg relative "
          style={{
            backgroundImage: "url(/assets/footer/footer-grid.svg)",
            backgroundSize: "cover",
            backgroundPosition: "right -60px top -160px",
            backgroundRepeat: "no-repeat",
          }}
        >
          <Logo className="text-white w-20 mb-8" />
          <div className="flex flex-wrap gap-8 lg:gap-12 xl:gap-20">
            {footerLinks.map((footerLink, i) => (
              <div className=" lg:w-auto  flex-shrink-0" key={i}>
                <h4 className="text-slate-400 text-lg font-semibold mb-4">
                  {footerLink.name}
                </h4>
                <ul className="flex flex-col gap-4">
                  {footerLink.links.map((link, j) => (
                    <li key={j}>
                      <a
                        className="text-white flex group gap-1 hover:text-indigo-400 transition-all"
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
              <h4 className="text-slate-400 flex-wrap text-lg font-semibold mb-4">
                Community
              </h4>
              <ul className="flex flex-col gap-4">
                <li>
                  <a
                    className="text-white flex items-center group gap-2 hover:text-indigo-400 transition-all"
                    href="https://discord.gg/EuesV2ZSnX"
                  >
                    <Discord />
                    Discord
                  </a>
                </li>
                <li>
                  <a
                    className="text-white flex items-center group gap-2 hover:text-indigo-400 transition-all"
                    href="https://github.com/inngest/inngest-js"
                  >
                    <Github />
                    GitHub
                  </a>
                </li>
                <li>
                  <a
                    className="text-white flex items-center group gap-2 hover:text-indigo-400 transition-all"
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
      <ul className="max-w-[1800px] m-auto py-8 lg:py-0 px-10 flex lg:gap-6 flex-col-reverse items-start lg:flex-row">
        <li className=" text-sm text-center py-1.5 lg:py-8 text-slate-300 font-medium">
          &copy;
          {new Date().getFullYear()} Inngest Inc.
        </li>
        <li className=" text-sm text-center">
          <a
            className="text-slate-400 py-1.5 lg:py-8 block hover:text-indigo-400 transition-colors"
            href="/privacy?ref=footer"
          >
            Privacy
          </a>
        </li>
        <li className=" text-sm text-center">
          <a
            className="text-slate-400 py-1.5 lg:py-8 block hover:text-indigo-400 transition-colors"
            href="/terms?ref=footer"
          >
            Terms and Conditions
          </a>
        </li>
        <li className=" text-sm text-center">
          <a
            className="text-slate-400 py-1.5 lg:py-8 block hover:text-indigo-400 transition-colors"
            href="/security?ref=footer"
          >
            Security
          </a>
        </li>
      </ul>
    </footer>
  );
}
