import Discord from "../Icons/Discord";
import Github from "../Icons/Github";
import Container from "../layout/Container";
import { ChevronRightIcon } from "@heroicons/react/20/solid";

export default function SocialCTA() {
  return (
    <div>
      <Container className="my-10 lg:mt-20 lg:mb-12 flex flex-col md:flex-row gap-8 lg:gap-12">
        <div className="w-full lg:w-1/2 ">
          <a
            href="https://www.inngest.com/discord"
            className="bg-[#5765f2] rounded flex items-center justify-center  text-white h-[140px] lg:h-[200px] hover:opacity-80 transition-all duration-150"
          >
            <Discord size="4em" />
          </a>
          <h4 className="lg:px-8 mt-4 lg:mt-8 mb-2 text-lg text-white">
            Join our Discord community
          </h4>
          <p className="lg:px-8 text-slate-400 text-sm">
            Join our Discord community to share feedback, get updates, and have
            a direct line to shaping the future of the SDK!
          </p>
          <a
            href="https://www.inngest.com/discord"
            className="group lg:mx-6  mt-3 lg:mt-6 inline-flex rounded-md text-sm font-medium px-6 py-2 bg-slate-800 hover:bg-slate-700 transition-all text-white"
          >
            Join the Community
            <ChevronRightIcon className="h-5 group-hover:translate-x-1 relative top-px transition-transform duration-150" />
          </a>
        </div>
        <div className="w-full lg:w-1/2">
          <a
            href="https://github.com/inngest/inngest"
            className="bg-slate-800 rounded flex items-center justify-center text-white h-[140px] lg:h-[200px]  hover:opacity-80 transition-all duration-150"
          >
            <Github size="4em" />
          </a>
          <h4 className="lg:px-8 mt-4 lg:mt-8 mb-2 text-lg text-white">
            Open Source
          </h4>
          <p className="lg:px-8 text-slate-400 text-sm">
            Inngest's core is open source, giving you piece of mind.
          </p>
          <a
            href="https://github.com/inngest/inngest"
            className="group lg:mx-6  mt-3 lg:mt-6 inline-flex rounded-md text-sm font-medium px-6 py-2 bg-slate-800 hover:bg-slate-700 transition-all text-white"
          >
            View Project
            <ChevronRightIcon className="h-5 group-hover:translate-x-1 relative top-px transition-transform duration-150" />
          </a>
        </div>
      </Container>
    </div>
  );
}
