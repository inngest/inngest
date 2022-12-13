import ArrowRight from "../Icons/ArrowRight";

function getType(type) {
  switch (type) {
    case "GUIDE":
      return {
        label: "Guide",
        icon: "",
      };
    case "TUTORIAL":
      return {
        label: "Tutorial",
        icon: "",
      };
    case "PATTERN":
      return {
        label: "Pattern",
        icon: "",
      };
    case "DOCS":
      return {
        label: "Docs",
        icon: "",
      };
    case "BLOG":
      return {
        label: "Blog",
        icon: "",
      };
    default:
      return {
        label: "Docs",
        icon: "",
      };
  }
}

export default function Learning({ type, href, title, description }) {
  const learningType = getType(type.toUpperCase());

  return (
    <a
      href={href}
      className="group/learning bg-slate-800/60 hover:bg-slate-800 p-4 xl:p-6 rounded-lg transition-all"
    >
      <span className="mb-2 font-semibold text-sm text-white">{type}</span>
      <h4 className="text-white mb-1.5 lg:mb-2.5 mt-3 text-lg lg:text-xl">
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
