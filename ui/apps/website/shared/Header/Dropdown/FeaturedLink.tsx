import ArrowRight from "../../Icons/ArrowRight";

export default function FeaturedLink({ link }) {
  return (
    <a
      href={link.url}
      className="hover:bg-slate-800/80 px-5 py-2 transition-all duration-150 flex items-start lg:items-center mb-1.5 leading-none group/nav-item"
    >
      <div
        className={`h-11 w-11 flex flex-shrink-0 items-center justify-center rounded ${link.iconBg}`}
      >
        <link.icon size={32} />
      </div>
      <div className="pl-3.5">
        <h4 className={`text-base text-white flex items-center`}>
          {link.title}
          <ArrowRight
            className="ml-0.5 group-hover/nav-item:translate-x-1 transition-all"
            width="w-6"
            height="h-6"
          />
        </h4>
        <span className="text-slate-400 text-sm">{link.desc}</span>
      </div>
    </a>
  );
}
