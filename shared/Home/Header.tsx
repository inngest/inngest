import Link from "next/link";

export default function Header() {
  return (
    <header className="fixed backdrop-blur-sm top-0 left-0 right-0 max-w-container-desktop m-auto flex justify-between items-center py-5 px-10 z-50">
      <div className="flex">
        <h1 className="mr-8">inngest</h1>
        <nav>
          <ul className="flex">
            <li>
              <Link href="roduct">
                <a className="px-5 py-2 text-sm">Product</a>
              </Link>
            </li>
            <li>
              <Link href="roduct">
                <a className="px-5 py-2 text-sm">Learn</a>
              </Link>
            </li>
            <li>
              <Link href="roduct">
                <a className="px-5 py-2 text-sm">Pricing</a>
              </Link>
            </li>
            <li>
              <Link href="roduct">
                <a className="px-5 py-2 text-sm">Blog</a>
              </Link>
            </li>
          </ul>
        </nav>
      </div>
      <div className="flex gap-6 items-center">
        <Link className="text-sm" href="https://app.inngest.com/login">
          <a className="text-sm">Log In</a>
        </Link>
        <Link href="sign-up">
          <a className="text-sm border border-slate-800 rounded-full px-5 py-2">
            Sign Up
          </a>
        </Link>
      </div>
    </header>
  );
}
