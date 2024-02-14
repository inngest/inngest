import ArrowRight from '../Icons/ArrowRight';
import { IconBlog, IconDocs, IconGuide, IconPatterns, IconTutorial } from '../Icons/duotone';

function getType(type) {
  switch (type) {
    case 'GUIDE':
      return {
        label: 'Guide',
        icon: IconGuide,
      };
    case 'TUTORIAL':
      return {
        label: 'Tutorial',
        icon: IconTutorial,
      };
    case 'PATTERN':
      return {
        label: 'Pattern',
        icon: IconPatterns,
      };
    case 'DOCS':
      return {
        label: 'Docs',
        icon: IconDocs,
      };
    case 'BLOG':
      return {
        label: 'Blog',
        icon: IconBlog,
      };
    default:
      return {
        label: 'Docs',
        icon: IconDocs,
      };
  }
}

export default function Learning({ type, href, title, description }) {
  const learningType = getType(type.toUpperCase());

  return (
    <a
      href={href}
      className="group/learning rounded-lg bg-slate-800/60 p-4 pt-4  transition-all hover:bg-slate-800 xl:p-6 xl:pt-5"
    >
      <span className="flex items-center gap-1 text-sm font-semibold text-white">
        <learningType.icon size={24} color="indigo" /> {type}
      </span>
      <h4 className="mb-1.5 mt-2 text-lg text-white lg:mb-2.5 lg:text-xl">{title}</h4>
      <p className="transition-color text-sm leading-6 text-indigo-200 group-hover/learning:text-white">
        {description}
      </p>

      <span className="transition-color mt-4 flex items-center text-sm font-medium text-indigo-400 group-hover/learning:text-white">
        Read {learningType.label.toLowerCase()}{' '}
        <ArrowRight className="ml-1 transition-transform group-hover/learning:translate-x-2" />
      </span>
    </a>
  );
}
