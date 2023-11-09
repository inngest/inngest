import { useState, useEffect } from "react";
import ArrowRight from "../Icons/ArrowRight";
import Logo from "../Icons/Logo";
import classNames from "src/utils/classNames";
import Github from "../Icons/Github";
import Discord from "../Icons/Discord";
import XSocialIcon from "../Icons/XSocialIcon";
import Container from "../layout/Container";
import BurgerMenu from "../Icons/BurgerMenu";
import CloseMenu from "../Icons/CloseMenu";
import HeaderDropdown from "./Dropdown";
import { productLinks, learnLinks } from "./headerLinks";

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
        scroll ? `bg-slate-1000/80 ` : "",
        `sticky backdrop-blur top-0 left-0 right-0 z-[100] transition-colors duration-200`
      )}
    >
      <Container className="flex justify-between items-center px-0">
        <div className="flex items-center w-full">
          <div
            className={classNames(
              menuState ? `bg-slate-900` : ``,
              `md:bg-transparent flex items-center py-5 md:py-0 w-full md:w-auto px-8 md:px-0 justify-between`
            )}
          >
            <a href="/" className="mr-4">
              <Logo className="text-white w-20 relative top-[2px]" />
            </a>
            <button
              className="text-slate-400 md:hidden"
              onClick={() => toggleMenu()}
            >
              {menuState ? <CloseMenu /> : <BurgerMenu />}
            </button>
          </div>
          <nav
            className={classNames(
              menuState ? `block` : `hidden`,
              `overflow-y-scroll md:overflow-visible w-full fixed bottom-0 md:bottom-auto -z-10 md:z-0 pt-[76px] md:pt-0 h-screen md:h-auto max-h-screen top-[0] left-0  right-0  bg-slate-900 md:bg-transparent md:static md:flex`
            )}
          >
            <div className="flex flex-col md:flex-row items-start md:items-center w-full">
              <ul className="flex flex-col md:flex-row md:items-center gap-4 md:gap-0">
                <li className="relative flex items-center group text-white font-medium md:px-5 md:py-8 text-sm">
                  <span className="hidden md:block group-hover:md:opacity-40 transition-opacity cursor-pointer">
                    Product
                  </span>
                  <HeaderDropdown navLinks={productLinks} />
                </li>
                <li className="relative flex flex-col md:flex-row md:items-center group text-white font-medium md:px-5 md:py-8 text-sm">
                  <span className="hidden md:block md:group-hover:opacity-40 transition-opacity cursor-pointer">
                    Learn
                  </span>
                  <HeaderDropdown navLinks={learnLinks} />
                </li>
                <li>
                  <a
                    href="/pricing?ref=nav"
                    className="mt-4 md:mt-0 flex items-center text-white font-medium px-7 md:px-5 py-2 text-sm hover:opacity-60"
                  >
                    Pricing
                  </a>
                </li>
                <li>
                  <a
                    href="/blog?ref=nav"
                    className="flex items-center text-white font-medium px-7 md:px-5 py-2 text-sm hover:opacity-60"
                  >
                    Blog
                  </a>
                </li>
                {/*
                <li>
                  <a
                    href="https://roadmap.inngest.com/roadmap?ref=nav"
                    target="_blank"
                    className="flex md:hidden lg:flex items-center text-white font-medium px-7 md:px-5 py-2 text-sm hover:opacity-60"
                  >
                    Roadmap
                  </a>
                </li>
                */}
                <li>
                  <a
                    href="https://roadmap.inngest.com/changelog?ref=nav"
                    target="_blank"
                    className="flex md:hidden lg:flex items-center text-white font-medium px-7 md:px-5 py-2 text-sm hover:opacity-60"
                  >
                    Changelog
                  </a>
                </li>
              </ul>
              <ul className="flex flex-shrink-0 md:items-center mt-6 md:mt-0 md:px-3 md:hidden xl:flex">
                <li>
                  <a
                    href="https://github.com/inngest/inngest"
                    className="flex flex-shrink-0 items-center text-white font-medium px-3.5 py-2 text-sm ml-4 hover:opacity-60"
                  >
                    <Github />
                  </a>
                </li>
                <li>
                  <a
                    href="https://www.inngest.com/discord"
                    className="flex flex-shrink-0 items-center text-white font-medium px-3.5 py-2 text-sm  hover:opacity-60"
                  >
                    <Discord />
                  </a>
                </li>
                <li>
                  <a
                    href="https://twitter.com/inngest"
                    className="flex flex-shrink-0 items-center text-white font-medium px-3.5 py-2 text-sm hover:opacity-60"
                  >
                    <XSocialIcon />
                  </a>
                </li>
              </ul>
            </div>
            <div className="px-8 md:px-10 py-8 md:py-0 flex gap-6 items-center md:w-1/3 md:justify-end flex-shrink-0">
              <a
                href={`${process.env.NEXT_PUBLIC_SIGNIN_URL}?ref=nav`}
                className="text-white font-medium text-sm  hover:opacity-60 duration-150 transition-all flex-shrink-0"
              >
                Sign In
              </a>

              <a
                href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=nav`}
                className="group flex gap-0.5 items-center rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-indigo-500 hover:bg-indigo-400 transition-all text-white flex-shrink-0"
              >
                Sign Up
                <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
              </a>
            </div>
          </nav>
        </div>
      </Container>
    </header>
  );
}
