import ArrowRight from '../../Icons/ArrowRight';
import FeaturedLink from './FeaturedLink';
import NavLink from './NavLink';

export default function HeaderDropdown({ navLinks }) {
  return (
    <div className="-left-4 top-[70px] px-2 md:absolute md:hidden md:overflow-auto md:rounded-lg md:bg-slate-950 md:px-0 md:shadow-2xl group-hover:md:block">
      <div className="flex flex-col md:w-[520px]">
        {!!navLinks.featured.length && (
          <div className="flex w-full flex-col">
            <h3 className="md:text-2xs mb-1 px-5 pt-3 text-base font-semibold text-slate-200 md:uppercase">
              {navLinks.featuredTitle}
            </h3>
            {navLinks.featured.map((link) => (
              <FeaturedLink key={link.title} link={link} />
            ))}
          </div>
        )}
        <div className="flex w-full flex-col">
          <h3 className="text-2xs mb-2 mt-3 px-5 font-semibold uppercase text-slate-400 md:text-slate-200">
            {navLinks.linksTitle}
          </h3>
          <div className="gri-cols-1 grid md:grid-cols-2 md:bg-slate-900">
            {navLinks.links.map((link, i) => (
              <NavLink
                key={i}
                theme={navLinks.linksTheme}
                link={link}
                len={navLinks.links.length}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
