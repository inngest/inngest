import Link from "next/link";
import ArrowRight from "../Icons/ArrowRight";

export default function Header() {
  return (
    <header className="fixed backdrop-blur-sm top-0 left-0 right-0 z-50">
      <div className="max-w-container-desktop m-auto flex justify-between items-center py-5 px-10">
        <div className="flex">
          <h1 className="mr-8">inngest</h1>
          <nav>
            <ul className="flex">
              <li>
                <a className="text-white font-medium px-5 py-2 text-sm">
                  Product
                </a>
              </li>
              <li>
                <a className="text-white font-medium px-5 py-2 text-sm">
                  Learn
                </a>
              </li>
              <li>
                <a className="text-white font-medium px-5 py-2 text-sm">
                  Pricing
                </a>
              </li>
              <li>
                <a className="text-white font-medium px-5 py-2 text-sm">Blog</a>
              </li>
            </ul>
          </nav>
        </div>
        <div className="flex gap-6 items-center">
          <a
            href="https://app.inngest.com/login"
            className="text-white font-medium text-sm"
          >
            Log In
          </a>

          <a
            href="sign-up"
            className="flex gap-0.5 items-center rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-indigo-500 hover:bg-indigo-400 transition-all text-white"
          >
            Sign Up
            <ArrowRight />
          </a>
        </div>
      </div>
    </header>
  );
}
