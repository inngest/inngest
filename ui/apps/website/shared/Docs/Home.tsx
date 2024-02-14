import Link from "next/link";
import { BoltIcon } from "@heroicons/react/24/outline";
import Logo from "../Icons/Logo";

export function Home() {
  return (
    <header className="max-w-none pt-24 pb-16 md:py-24 px-8 sm:px-10 lg:px-16 bg-white dark:bg-slate-900/50 rounded-lg border border-slate-900/7.5 bg-[url(/assets/textures/wave-gray.svg)] bg-cover">
      <h1>Inngest Documentation</h1>
      <p className="mb-0 font-medium text-slate-500 dark:text-slate-400">
        Learn how to build with Inngest's durable workflow engine.
      </p>
    </header>
  );
}
