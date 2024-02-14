import Link from 'next/link';
import { BoltIcon } from '@heroicons/react/24/outline';

import Logo from '../Icons/Logo';

export function Home() {
  return (
    <header className="border-slate-900/7.5 max-w-none rounded-lg border bg-white bg-[url(/assets/textures/wave-gray.svg)] bg-cover px-8 pb-16 pt-24 sm:px-10 md:py-24 lg:px-16 dark:bg-slate-900/50">
      <h1>Inngest Documentation</h1>
      <p className="mb-0 font-medium text-slate-500 dark:text-slate-400">
        Learn how to build with Inngest's durable workflow engine.
      </p>
    </header>
  );
}
