import { ChevronRightIcon } from '@heroicons/react/20/solid';

import Discord from '../Icons/Discord';
import Github from '../Icons/Github';
import Container from '../layout/Container';

export default function SocialCTA() {
  return (
    <div>
      <Container className="my-10 flex flex-col gap-8 md:flex-row lg:mb-12 lg:mt-20 lg:gap-12">
        <div className="w-full lg:w-1/2 ">
          <a
            href="https://www.inngest.com/discord"
            className="flex h-[140px] items-center justify-center rounded  bg-[#5765f2] text-white transition-all duration-150 hover:opacity-80 lg:h-[200px]"
          >
            <Discord size="4em" />
          </a>
          <h4 className="mb-2 mt-4 text-lg text-white lg:mt-8 lg:px-8">
            Join our Discord community
          </h4>
          <p className="text-sm text-slate-400 lg:px-8">
            Join our Discord community to share feedback, get updates, and have a direct line to
            shaping the future of the SDK!
          </p>
          <a
            href="https://www.inngest.com/discord"
            className="group mt-3  inline-flex rounded-md bg-slate-800 px-6 py-2 text-sm font-medium text-white transition-all hover:bg-slate-700 lg:mx-6 lg:mt-6"
          >
            Join the Community
            <ChevronRightIcon className="relative top-px h-5 transition-transform duration-150 group-hover:translate-x-1" />
          </a>
        </div>
        <div className="w-full lg:w-1/2">
          <a
            href="https://github.com/inngest/inngest"
            className="flex h-[140px] items-center justify-center rounded bg-slate-800 text-white transition-all  duration-150 hover:opacity-80 lg:h-[200px]"
          >
            <Github size="4em" />
          </a>
          <h4 className="mb-2 mt-4 text-lg text-white lg:mt-8 lg:px-8">Open Source</h4>
          <p className="text-sm text-slate-400 lg:px-8">
            Inngest's core is open source, giving you piece of mind.
          </p>
          <a
            href="https://github.com/inngest/inngest"
            className="group mt-3  inline-flex rounded-md bg-slate-800 px-6 py-2 text-sm font-medium text-white transition-all hover:bg-slate-700 lg:mx-6 lg:mt-6"
          >
            View Project
            <ChevronRightIcon className="relative top-px h-5 transition-transform duration-150 group-hover:translate-x-1" />
          </a>
        </div>
      </Container>
    </div>
  );
}
