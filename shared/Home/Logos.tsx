import Link from "next/link";
import Image from "next/image";
import clsx from "clsx";

import Container from "../layout/Container";

export default function Logos({
  heading,
  logos,
  variant = "dark",
}: {
  heading: string | React.ReactNode;
  logos: { src: string; name: string; href?: string }[];
  variant?: "dark" | "light";
}) {
  const hasLinks = !!logos.find((l) => !!l.href);
  return (
    <Container className="my-36 mx-auto max-w-4xl">
      <h2
        className={clsx(
          "text-lg tracking-tight text-center",
          variant === "dark" && "text-slate-400 drop-shadow",
          variant === "light" && "text-slate-700"
        )}
      >
        {heading}
      </h2>
      <div
        className={clsx(
          "mt-10 flex flex-wrap lg:flex-row gap-y-8 items-center justify-center",
          hasLinks ? "gap-x-4" : "gap-x-16"
        )}
      >
        {logos.map(({ src, name, href }, idx) => {
          if (href) {
            return (
              <Link
                href={href}
                className={clsx(
                  "group flex items-center justify-center h-16 w-40 px-6 py-6 rounded-lg border transition-all",
                  variant === "dark" &&
                    "border-slate-900 hover:border-slate-700",
                  variant === "light" &&
                    "border-slate-200 hover:border-slate-300"
                )}
              >
                <Image
                  key={idx}
                  src={src}
                  alt={name}
                  width={120}
                  height={30}
                  className="text-white max-h-[40px] pointer-events-none opacity-80 transition-all group-hover:opacity-100 grayscale group-hover:grayscale-0"
                />
              </Link>
            );
          }
          return (
            <Image
              key={idx}
              src={src}
              alt={name}
              title={name}
              width={120}
              height={30}
              className="text-white max-h-[36px] transition-all grayscale hover:grayscale-0"
            />
          );
        })}
      </div>
    </Container>
  );
}
