import { useEffect, useState } from 'react';
import classNames from 'src/utils/classNames';

import ArrowRight from '../Icons/ArrowRight';
import BurgerMenu from '../Icons/BurgerMenu';
import CloseMenu from '../Icons/CloseMenu';
import Discord from '../Icons/Discord';
import Github from '../Icons/Github';
import Logo from '../Icons/Logo';
import XSocialIcon from '../Icons/XSocialIcon';
import Container from '../layout/Container';
import HeaderDropdown from './Dropdown';
import { learnLinks, productLinks } from './headerLinks';

export default function Header() {
  const [scroll, setScroll] = useState(false);
  const [menuState, setMenuState] = useState(false);

  useEffect(() => {
    window.addEventListener('scroll', () => {
      setScroll(window.scrollY > 40);
    });
  }, []);

  const toggleMenu = () => {
    setMenuState(!menuState);
  };

  return (
    <header
      className={classNames(
        scroll ? `bg-slate-1000/80 ` : '',
        `sticky left-0 right-0 top-0 z-[100] backdrop-blur transition-colors duration-200`
      )}
    >
      <Container className="flex items-center justify-between px-0">
        <div className="flex w-full items-center">
          <div
            className={classNames(
              menuState ? `bg-slate-900` : ``,
              `flex w-full items-center justify-between px-8 py-5 md:w-auto md:bg-transparent md:px-0 md:py-0`
            )}
          >
            <a href="/" className="mr-4">
              <Logo className="relative top-[2px] w-20 text-white" />
            </a>
            <button className="text-slate-400 md:hidden" onClick={() => toggleMenu()}>
              {menuState ? <CloseMenu /> : <BurgerMenu />}
            </button>
          </div>
          <nav
            className={classNames(
              menuState ? `block` : `hidden`,
              `fixed bottom-0 left-0 right-0 top-[0] -z-10 h-screen max-h-screen w-full overflow-y-scroll bg-slate-900 pt-[76px] md:static md:bottom-auto md:z-0  md:flex  md:h-auto md:overflow-visible md:bg-transparent md:pt-0`
            )}
          >
            <div className="flex w-full flex-col items-start md:flex-row md:items-center">
              <ul className="flex flex-col gap-4 md:flex-row md:items-center md:gap-0">
                <li className="group relative flex items-center text-sm font-medium text-white md:px-5 md:py-8">
                  <span className="hidden cursor-pointer transition-opacity md:block group-hover:md:opacity-40">
                    Product
                  </span>
                  <HeaderDropdown navLinks={productLinks} />
                </li>
                <li className="group relative flex flex-col text-sm font-medium text-white md:flex-row md:items-center md:px-5 md:py-8">
                  <span className="hidden cursor-pointer transition-opacity md:block md:group-hover:opacity-40">
                    Docs
                  </span>
                  <HeaderDropdown navLinks={learnLinks} />
                </li>
                <li>
                  <a
                    href="/customers?ref=nav"
                    className="flex items-center px-7 py-2 text-sm font-medium text-white hover:opacity-60 md:hidden md:px-5 lg:flex"
                  >
                    Case Studies
                  </a>
                </li>
                <li>
                  <a
                    href="/pricing?ref=nav"
                    className="mt-4 flex items-center px-7 py-2 text-sm font-medium text-white hover:opacity-60 md:mt-0 md:px-5"
                  >
                    Pricing
                  </a>
                </li>
                <li>
                  <a
                    href="/blog?ref=nav"
                    className="flex items-center px-7 py-2 text-sm font-medium text-white hover:opacity-60 md:px-5"
                  >
                    Blog
                  </a>
                </li>
              </ul>
              <ul className="mt-6 flex flex-shrink-0 md:mt-0 md:hidden md:items-center md:px-3 xl:flex">
                <li>
                  <a
                    href="https://github.com/inngest/inngest"
                    className="ml-4 flex flex-shrink-0 items-center px-3.5 py-2 text-sm font-medium text-white hover:opacity-60"
                  >
                    <Github />
                  </a>
                </li>
                <li>
                  <a
                    href="https://www.inngest.com/discord"
                    className="flex flex-shrink-0 items-center px-3.5 py-2 text-sm font-medium text-white  hover:opacity-60"
                  >
                    <Discord />
                  </a>
                </li>
                <li>
                  <a
                    href="https://twitter.com/inngest"
                    className="flex flex-shrink-0 items-center px-3.5 py-2 text-sm font-medium text-white hover:opacity-60"
                  >
                    <XSocialIcon />
                  </a>
                </li>
              </ul>
            </div>
            <div className="flex flex-shrink-0 items-center gap-6 px-8 py-8 md:w-1/3 md:justify-end md:px-10 md:py-0">
              <a
                href={`${process.env.NEXT_PUBLIC_SIGNIN_URL}?ref=nav`}
                className="flex-shrink-0 text-sm font-medium  text-white transition-all duration-150 hover:opacity-60"
              >
                Sign In
              </a>

              <a
                href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=nav`}
                className="group flex flex-shrink-0 items-center gap-0.5 rounded-full bg-indigo-500 py-2 pl-6 pr-5  text-sm font-medium text-white transition-all hover:bg-indigo-400"
              >
                Sign Up
                <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
              </a>
            </div>
          </nav>
        </div>
      </Container>
    </header>
  );
}
