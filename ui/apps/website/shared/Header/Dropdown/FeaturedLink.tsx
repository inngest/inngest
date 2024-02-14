import ArrowRight from '../../Icons/ArrowRight';

export default function FeaturedLink({ link }) {
  return (
    <a
      href={link.url}
      className="group/nav-item mb-1.5 flex items-start px-5 py-2 leading-none transition-all duration-150 hover:bg-slate-800/80 lg:items-center"
    >
      <div
        className={`flex h-11 w-11 flex-shrink-0 items-center justify-center rounded ${link.iconBg}`}
      >
        <link.icon size={32} />
      </div>
      <div className="pl-3.5">
        <h4 className={`flex items-center text-base text-white`}>
          {link.title}
          <ArrowRight
            className="ml-0.5 transition-all group-hover/nav-item:translate-x-1"
            width="w-6"
            height="h-6"
          />
        </h4>
        <span className="text-sm text-slate-400">{link.desc}</span>
      </div>
    </a>
  );
}
