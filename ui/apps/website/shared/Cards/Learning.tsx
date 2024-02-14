import ArrowRight from "../Icons/ArrowRight";
import {
  IconDocs,
  IconPatterns,
  IconGuide,
  IconTutorial,
  IconBlog,
} from "../Icons/duotone";

function getType(type) {
  switch (type) {
    case "GUIDE":
      return {
        label: "Guide",
        icon: IconGuide,
      };
    case "TUTORIAL":
      return {
        label: "Tutorial",
        icon: IconTutorial,
      };
    case "PATTERN":
      return {
        label: "Pattern",
        icon: IconPatterns,
      };
    case "DOCS":
      return {
        label: "Docs",
        icon: IconDocs,
      };
    case "BLOG":
      return {
        label: "Blog",
        icon: IconBlog,
      };
    default:
      return {
        label: "Docs",
        icon: IconDocs,
      };
  }
}

export default function Learning({ type, href, title, description }) {
  const learningType = getType(type.toUpperCase());

  return (
    <a
      href={href}
      className="group/learning bg-slate-800/60 hover:bg-slate-800 p-4 pt-4  xl:p-6 xl:pt-5 rounded-lg transition-all"
    >
      <span className="font-semibold text-sm text-white flex items-center gap-1">
        <learningType.icon size={24} color="indigo" /> {type}
      </span>
      <h4 className="text-white mb-1.5 lg:mb-2.5 mt-2 text-lg lg:text-xl">
        {title}
      </h4>
      <p className="text-indigo-200 group-hover/learning:text-white transition-color text-sm leading-6">
        {description}
      </p>

      <span className="group-hover/learning:text-white flex items-center text-indigo-400 font-medium text-sm mt-4 transition-color">
        Read {learningType.label.toLowerCase()}{" "}
        <ArrowRight className="transition-transform ml-1 group-hover/learning:translate-x-2" />
      </span>
    </a>
  );
}
