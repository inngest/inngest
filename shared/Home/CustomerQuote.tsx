import Image from "next/image";
import Link from "next/link";
import clsx from "clsx";
import { ChevronRightIcon } from "@heroicons/react/20/solid";

const MeshGradient = `
radial-gradient(at 21% 4%, hsla(209,94%,39%,0.69) 0px, transparent 50%),
radial-gradient(at 97% 96%, hsla(279,100%,76%,0.53) 0px, transparent 50%),
radial-gradient(at 43% 61%, hsla(279,100%,76%,0.53) 0px, transparent 50%),
radial-gradient(at 73% 24%, hsla(279,100%,76%,0.7) 0px, transparent 50%),
radial-gradient(at 5% 94%, hsla(210,75%,64%,0.81) 0px, transparent 50%),
url(/assets/textures/wave.svg)
`;

const MeshGradientLight = `
radial-gradient(at 21% 4%, hsla(209,94%,39%,0.07) 0px, transparent 50%),
radial-gradient(at 97% 96%, hsla(279,100%,76%,0.053) 0px, transparent 50%),
radial-gradient(at 43% 61%, hsla(279,100%,76%,0.053) 0px, transparent 50%),
radial-gradient(at 73% 24%, hsla(279,100%,76%,0.07) 0px, transparent 50%),
radial-gradient(at 5% 94%, hsla(210,75%,64%,0.081) 0px, transparent 50%),
url(/assets/textures/wave.svg)
`;

export default function CustomerQuote({
  quote,
  name,
  avatar,
  logo,
  className,
  variant = "dark",
  cta,
}: {
  quote: string;
  name: string;
  className?: string;
  variant?: "dark" | "light";
  avatar?: string;
  logo?: string;
  cta?: { href: string; text: string };
}) {
  return (
    <aside
      className={clsx(
        "p-2.5 relative bg-slate-100/10 rounded-[16px] backdrop-blur",
        className
      )}
    >
      <div
        className="relative z-10 py-5 px-8 flex flex-col items-start gap-2 rounded-lg shadow"
        style={{
          backgroundColor:
            variant === "dark"
              ? `hsla(235,79%,63%,1)`
              : "hsla(235,79%,63%,0.2)",
          backgroundImage:
            variant === "dark" ? MeshGradient : MeshGradientLight,
        }}
      >
        <div
          className={clsx(
            "text-sm mb-2 md:text-base lg:text-lg font-medium",
            variant === "dark" ? "text-white drop-shadow" : "text-slate-900"
          )}
        >
          &ldquo;{quote}&rdquo;
        </div>
        <div
          className={clsx(
            "flex flex-row gap-4 w-full items-center text-base font-medium",
            variant === "dark" ? "text-indigo-50 drop-shadow" : "text-slate-800"
          )}
        >
          {avatar && (
            <Image
              src={avatar}
              alt={`Image of ${name}`}
              height={36}
              width={36}
              className="rounded-full"
            />
          )}
          <span className="grow">{name}</span>
          {logo && (
            <Image
              src={logo}
              alt={`${name}'s company logo`}
              height={36}
              width={110}
            />
          )}
        </div>
        {cta && (
          <Link
            href={cta.href}
            className="group self-end flex flex-row items-center gap-0.5 mt-3 py-1.5 pl-3 pr-1.5 border border-white/50 rounded-lg text-sm text-indigo-50 transition-all hover:bg-white/10 font-medium whitespace-nowrap"
          >
            {cta.text}
            <ChevronRightIcon className="h-4 transition-all group-hover:translate-x-0.5" />
          </Link>
        )}
      </div>
    </aside>
  );
}
