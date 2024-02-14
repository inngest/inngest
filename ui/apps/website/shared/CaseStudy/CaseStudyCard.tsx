import Image from 'next/image';
import Link from 'next/link';

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
    <Link href={href} className="group flex h-full text-slate-50">
      <div className="flex grow flex-col justify-items-start rounded-2xl border border-slate-100/10 bg-slate-800/10 bg-[url(/assets/textures/wave.svg)] bg-contain p-8 transition-all group-hover:border-slate-100/20 group-hover:bg-slate-800/30">
        <div className="mb-4 text-sm font-medium text-slate-500">Case Study</div>
        <h2 className="text-2xl font-bold">{title}</h2>
        <div className="space-between my-10 flex min-h-20 grow flex-row items-center gap-8 md:flex-col lg:h-24 lg:flex-row">
          <p className="text-slate-300">{snippet}</p>
          <Image
            src={logo}
            alt={`${name} logo`}
            title={name}
            width={240 * 0.6 * 1}
            height={120 * 0.6 * 1}
            className="width-auto mx-auto w-36 shrink-0 text-white transition-all"
          />
        </div>
        <div className="flex items-end justify-between">
          <div className="flex gap-2 text-sm font-medium text-slate-500">
            {tags.map((tag) => (
              <span>{tag}</span>
            ))}
          </div>
          <div>
            Read the case study{' '}
            <span className="ml-1 transition-all group-hover:translate-x-0.5">→</span>
          </div>
        </div>
      </div>
    </Link>
  );
}
