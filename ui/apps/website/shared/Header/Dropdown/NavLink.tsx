import ArrowRight from '../../Icons/ArrowRight';

export default function NavLink({ theme, link, len }) {
  const borderPattern =
    len % 2 === 0
      ? 'lg:[&:nth-last-child(-n+2)]:border-b-transparent'
      : 'lg:[&:last-child]:border-b-transparent';

  return (
    <a
      href={link.url}
      className={`group/nav-item flex items-center border-dashed border-slate-800 py-2.5 pl-4 pr-5 text-slate-200 transition-all duration-150 hover:bg-slate-800/60 hover:text-white lg:border-b lg:bg-slate-900 lg:odd:border-r ${borderPattern}`}
    >
      <link.icon size={28} color={theme} />
      <span className="ml-1.5">{link.title}</span>
      <ArrowRight className="ml-1 text-slate-400 transition-all group-hover/nav-item:translate-x-1 group-hover/nav-item:text-white" />
    </a>
  );
}
