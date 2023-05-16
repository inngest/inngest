import FeaturedLink from "./FeaturedLink";
import ArrowRight from "../../Icons/ArrowRight";
import NavLink from "./NavLink";

export default function HeaderDropdown({ navLinks }) {
  return (
    <div className="px-2 md:px-5 pb-4 md:overflow-auto md:bg-slate-950 md:rounded-lg md:absolute top-[70px] -left-4 md:hidden group-hover:md:block shadow-2xl">
      <div className="flex flex-col md:w-[520px]">
        {!!navLinks.featured.length && (
          <div className="flex w-full flex-col">
            <h3 className="text-base md:text-2xs md:uppercase text-slate-200 font-semibold mb-1 px-5 pt-3">
              {navLinks.featuredTitle}
            </h3>
            {navLinks.featured.map((link) => (
              <FeaturedLink key={link.title} link={link} />
            ))}
          </div>
        )}
        <div className="flex flex-col w-full">
          <h3 className="text-2xs uppercase text-slate-400 md:text-slate-200 font-semibold mb-2 mt-3 px-5">
            {navLinks.linksTitle}
          </h3>
          <div className="grid gri-cols-1 md:grid-cols-2 md:bg-slate-900">
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
