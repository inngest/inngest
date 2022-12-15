import ArrowRight from "../../Icons/ArrowRight";
export default function NavLink({ link, len }) {
  const borderPattern =
    len % 2 === 0
      ? "lg:[&:nth-last-child(-n+2)]:border-b-transparent"
      : "lg:[&:last-child]:border-b-transparent";

  return (
    <a
      href={link.url}
      className={`text-slate-200 lg:border-b border-slate-800 border-dashed lg:odd:border-r flex items-center py-3 pl-4 pr-5 lg:bg-slate-900 hover:bg-slate-800/60 hover:text-white transition-all duration-150 group/nav-item ${borderPattern}`}
    >
      <link.icon />
      <span className="ml-1.5">{link.title}</span>
      <ArrowRight className="ml-1 text-slate-400 group-hover/nav-item:translate-x-1 group-hover/nav-item:text-white transition-all" />
    </a>
  );
}
