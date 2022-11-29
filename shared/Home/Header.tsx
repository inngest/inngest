import Link from "next/link";
import { useState, useEffect } from "react";
import ArrowRight from "../Icons/ArrowRight";
import Logo from "../Icons/Logo";
import classNames from "src/utils/classNames";
import Github from "../Icons/Github";
import Discord from "../Icons/Discord";
import Twitter from "../Icons/Twitter";
import Container from "./Container";
import BurgerMenu from "../Icons/BurgerMenu";

export default function Header() {
  const [scroll, setScroll] = useState(false);
  const [menuState, setMenuState] = useState(false);

  useEffect(() => {
    window.addEventListener("scroll", () => {
      setScroll(window.scrollY > 40);
    });
  }, []);

  const toggleMenu = () => {
    setMenuState(!menuState);
  };

  return (
    <header
      className={classNames(
        scroll ? `bg-slate-950/80` : "",
        `sticky backdrop-blur-sm top-0 left-0 right-0 z-50 transition-colors duration-200`
      )}
    >
      <Container className="flex justify-between items-center  py-5 lg:py-0">
        <div className="flex  items-center">
          <a href="/" className="mr-4">
            <Logo className="text-white w-20 relative top-[2px]" />
          </a>
          <nav
            className={classNames(
              menuState ? `block` : `hidden`,
              `overflow-y-scroll lg:overflow-visible fixed bottom-0 lg:bottom-auto -z-10 lg:z-0 pt-[76px] lg:pt-0 h-screen lg:h-auto max-h-screen top-[0] left-0  right-0  bg-slate-900 lg:bg-transparent lg:static lg:flex`
            )}
          >
            <ul className="flex flex-col lg:flex-row lg:items-center">
              <li className="relative flex items-center group text-white font-medium lg:px-5 lg:py-8 text-sm">
                <span className="hidden lg:block group-hover:lg:opacity-40 transition-opacity cursor-pointer">
                  Product
                </span>
                <div className="overflow-hidden lg:overflow-auto group-hover:lg:bg-slate-800 lg:rounded-lg lg:absolute top-[70px] lg:hidden group-hover:lg:block">
                  <div className="flex flex-col md:flex-row md:w-[650px]">
                    <div className="flex w-full flex-col py-5 p-4">
                      <h3 className="text-sm text-slate-400 mb-1 px-4">
                        Product
                      </h3>
                      <a
                        href="/features/sdk?ref=nav"
                        className="hover:bg-slate-700/80 px-4 py-3 rounded  transition-all duration-150"
                      >
                        <h4 className="text-base text-white">
                          TypeScript & JavaScript SDK
                        </h4>
                        <span className="text-slate-400">
                          Event-driven and scheduled serverless functions
                        </span>
                      </a>
                      <a
                        href="/features/step-functions?ref=nav"
                        className="hover:bg-slate-700/80 px-4 py-3 rounded  transition-all duration-150"
                      >
                        <h4 className="text-base text-white">Step Functions</h4>
                        <span className="text-slate-400">
                          Build complex conditional workflows
                        </span>
                      </a>
                    </div>
                    <div className="lg:bg-slate-700 flex flex-col  md:w-[380px] px-6 py-4">
                      <h3 className="text-sm text-slate-400 mb-1 px-2">
                        Use Cases
                      </h3>
                      <div className="flex flex-col">
                        <a
                          href="/uses/serverless-cron-jobs?ref=nav"
                          className="text-white py-1.5 px-2 hover:bg-slate-800/60 rounded transition-all duration-150"
                        >
                          Scheduled & cron jobs
                        </a>
                        <a
                          href="/uses/serverless-node-background-jobs?ref=nav"
                          className="text-white py-1.5 px-2 hover:bg-slate-800/60 rounded transition-all duration-150"
                        >
                          Background tasks
                        </a>
                        <a
                          href="/uses/internal-tools?ref=nav"
                          className="text-white py-1.5 px-2 hover:bg-slate-800/60 rounded transition-all duration-150"
                        >
                          Internal tools
                        </a>
                        <a
                          href="/uses/user-journey-automation?ref=nav"
                          className="text-white py-1.5 px-2 hover:bg-slate-800/60 rounded transition-all duration-150"
                        >
                          User journey automation
                        </a>
                      </div>
                    </div>
                  </div>
                </div>
              </li>
              <li className="relative flex flex-col lg:flex-row lg:items-center group text-white font-medium lg:px-5 lg:py-8 text-sm">
                <span className="hidden lg:block lg:group-hover:opacity-40 transition-opacity cursor-pointer">
                  Learn
                </span>
                <div className="overflow-hidden lg:group-hover:bg-slate-800 lg:rounded-lg lg:absolute top-[70px] lg:hidden lg:group-hover:block">
                  <div className="flex flex-col md:flex-row lg:w-[460px]">
                    <div className="flex w-full flex-col py-5 p-4">
                      <h3 className="text-sm text-slate-400 mb-1 px-4">
                        Learn
                      </h3>
                      <a
                        href="/docs?ref=nav"
                        className="hover:bg-slate-700/80 px-4 py-3 rounded  transition-all duration-150"
                      >
                        <h4 className="text-base text-white">Docs</h4>
                        <span className="text-slate-400">
                          Event-driven and scheduled serverless functions
                        </span>
                      </a>
                      <a
                        href="/patterns?ref=nav"
                        className="hover:bg-slate-700/80 px-4 py-3 rounded  transition-all duration-150"
                      >
                        <h4 className="text-base text-white">
                          Patterns: Async and Event-driven
                        </h4>
                        <span className="text-slate-400">
                          How to build asynchronous functionality by example
                        </span>
                      </a>
                    </div>
                  </div>
                </div>
              </li>
              <li>
                <a
                  href="/pricing?ref=nav"
                  className="flex items-center text-white font-medium px-8 lg:px-5 py-2 text-sm  hover:opacity-60"
                >
                  Pricing
                </a>
              </li>
              <li>
                <a
                  href="/blog?ref=nav"
                  className="flex items-center text-white font-medium px-8 lg:px-5 py-2 text-sm  hover:opacity-60"
                >
                  Blog
                </a>
              </li>
            </ul>
            <ul className="flex lg:items-center mt-2 lg:mt-0">
              <li>
                <a
                  href="https://github.com/inngest/inngest"
                  className="flex items-center text-white font-medium px-3.5 py-2 text-sm ml-4  hover:opacity-60"
                >
                  <Github />
                </a>
              </li>
              <li>
                <a
                  href="https://discord.gg/EuesV2ZSnX"
                  className="flex items-center text-white font-medium px-3.5 py-2 text-sm  hover:opacity-60"
                >
                  <Discord />
                </a>
              </li>
              <li>
                <a
                  href="https://twitter.com/inngest"
                  className="flex items-center text-white font-medium px-3.5 py-2 text-sm hover:opacity-60"
                >
                  <Twitter />
                </a>
              </li>
            </ul>
          </nav>
        </div>
        <div className="flex gap-6 items-center">
          <a
            href="https://app.inngest.com/login?ref=nav"
            className="text-white font-medium text-sm"
          >
            Log In
          </a>

          <a
            href="/sign-up?ref=nav"
            className="group flex gap-0.5 items-center rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-indigo-500 hover:bg-indigo-400 transition-all text-white"
          >
            Sign Up
            <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
          </a>
          <button
            className="text-slate-400 xl:m lg:hidden"
            onClick={() => toggleMenu()}
          >
            <BurgerMenu />
          </button>
        </div>
      </Container>
    </header>
  );
}
