import Link from "next/link";
import { useState, useEffect } from "react";
import ArrowRight from "../Icons/ArrowRight";
import Logo from "../Icons/Logo";
import classNames from "src/utils/classnames";
import Github from "../Icons/Github";
import Discord from "../Icons/Discord";
import Twitter from "../Icons/Twitter";
import Container from "./Container";

export default function Header() {
  const [scroll, setScroll] = useState(false);

  useEffect(() => {
    window.addEventListener("scroll", () => {
      setScroll(window.scrollY > 40);
    });
  }, []);

  return (
    <header
      className={classNames(
        scroll ? `bg-slate-950/80` : "",
        `sticky backdrop-blur-sm top-0 left-0 right-0 z-50 transition-colors duration-200`
      )}
    >
      <Container className="flex justify-between items-center py-5">
        <div className="flex items-center">
          <h1 className="mr-4">
            <Logo className="text-white w-20 relative top-[2px]" />
          </h1>
          <nav>
            <ul className="flex items-center">
              <li>
                <a className="flex items-center text-white font-medium px-5 py-2 text-sm">
                  Product
                </a>
              </li>
              <li>
                <a className="flex items-center text-white font-medium px-5 py-2 text-sm">
                  Learn
                </a>
              </li>
              <li>
                <a className="flex items-center text-white font-medium px-5 py-2 text-sm">
                  Pricing
                </a>
              </li>
              <li>
                <a
                  href="/blog?ref=nav"
                  className="flex items-center text-white font-medium px-5 py-2 text-sm"
                >
                  Blog
                </a>
              </li>
              <li>
                <a className="flex items-center text-white font-medium px-3.5 py-2 text-sm ml-4">
                  <Github />
                </a>
              </li>
              <li>
                <a className="flex items-center text-white font-medium px-3.5 py-2 text-sm">
                  <Discord />
                </a>
              </li>
              <li>
                <a className="flex items-center text-white font-medium px-3.5 py-2 text-sm">
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
        </div>
      </Container>
    </header>
  );
}
