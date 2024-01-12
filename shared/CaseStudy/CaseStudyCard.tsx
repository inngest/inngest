import Link from "next/link";
import Image from "next/image";

export default function CaseStudyCard({
  href,
  title,
  snippet,
  name,
  logo,
  tags = [],
}: {
  href: string;
  title: string;
  snippet: React.ReactNode;
  name: string;
  logo: string;
  tags?: string[];
}) {
  return (
    <Link href={href} className="group text-slate-50 h-full flex">
      <div className="p-8 grow flex flex-col justify-items-start border border-slate-100/10 group-hover:border-slate-100/20 rounded-2xl transition-all bg-slate-800/10 group-hover:bg-slate-800/30 bg-[url(/assets/textures/wave.svg)] bg-contain">
        <div className="mb-4 text-sm font-medium text-slate-500">
          Case Study
        </div>
        <h2 className="font-bold text-2xl">{title}</h2>
        <div className="my-10 grow lg:grow-0 min-h-20 lg:h-24 flex flex-row md:flex-col lg:flex-row items-center space-between gap-8">
          <p className="text-slate-300">{snippet}</p>
          <Image
            src={logo}
            alt={`${name} logo`}
            title={name}
            width={240 * 0.6 * 1}
            height={120 * 0.6 * 1}
            className="w-36 shrink-0 text-white mx-auto width-auto transition-all"
          />
        </div>
        <div className="flex items-end justify-between">
          <div className="flex gap-2 text-sm font-medium text-slate-500">
            {tags.map((tag) => (
              <span>{tag}</span>
            ))}
          </div>
          <div>
            Read the case study{" "}
            <span className="ml-1 transition-all group-hover:translate-x-0.5">
              →
            </span>
          </div>
        </div>
      </div>
    </Link>
  );
}
