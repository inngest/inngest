import FeaturedLink from "./FeaturedLink";
import ArrowRight from "../../Icons/ArrowRight";
import NavLink from "./NavLink";

export default function HeaderDropdown({ navLinks }) {
  return (
    <div className="overflow-hidden px-2 md:px-5 lg:px-0 lg:overflow-auto lg:bg-slate-950 lg:rounded-lg lg:absolute top-[70px] -left-4 lg:hidden group-hover:lg:block">
      <div className="flex flex-col md:w-[520px]">
        <div className="flex w-full flex-col">
          <h3 className="text-base lg:text-2xs lg:uppercase text-slate-200 font-semibold mb-1 px-5 pt-3">
            {navLinks.featuredTitle}
          </h3>
          {navLinks.featured.map((link) => (
            <FeaturedLink key={link.title} link={link} />
          ))}
        </div>
        <div className="flex flex-col w-full">
          <h3 className="text-2xs uppercase text-slate-400 lg:text-slate-200 font-semibold mb-2 mt-3 px-5">
            {navLinks.linksTitle}
          </h3>
          <div className="grid gri-cols-1 md:grid-cols-2 lg:bg-slate-900">
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
