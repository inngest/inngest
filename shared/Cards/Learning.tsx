import ArrowRight from "../Icons/ArrowRight";
import {
  IconBackgroundTasks,
  IconDeploying,
  IconDocs,
  IconFunctions,
  IconJourney,
  IconPatterns,
  IconGuide,
  IconScheduled,
  IconSendEvents,
  IconSteps,
  IconTools,
  IconWritingFns,
} from "../Icons/duotone";

function getType(type) {
  switch (type) {
    case "GUIDE":
      return {
        label: "Guide",
        icon: <IconGuide size={20} color="indigo" />,
      };
    case "TUTORIAL":
      return {
        label: "Tutorial",
        icon: "",
      };
    case "PATTERN":
      return {
        label: "Pattern",
        icon: <IconPatterns size={32} color="indigo" />,
      };
    case "DOCS":
      return {
        label: "Docs",
        icon: <IconDocs size={32} color="indigo" />,
      };
    case "BLOG":
      return {
        label: "Blog",
        icon: "",
      };
    default:
      return {
        label: "Docs",
        icon: <IconDocs size={32} color="indigo" />,
      };
  }
}

export default function Learning({ type, href, title, description }) {
  const learningType = getType(type.toUpperCase());

  console.log(learningType);

  return (
    <a
      href={href}
      className="group/learning bg-slate-800/60 hover:bg-slate-800 p-4 pt-3  xl:p-6 xl:pt-4 rounded-lg transition-all"
    >
      <span className="font-semibold text-sm text-white flex items-center -ml-2">
        {learningType.icon} {type}
      </span>
      <h4 className="text-white mb-1.5 lg:mb-2.5 mt-1.5 text-lg lg:text-xl">
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
