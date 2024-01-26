import { IconElixir } from '@inngest/components/icons/languages/Elixir';
import { IconGo } from '@inngest/components/icons/languages/Go';
import { IconJavaScript } from '@inngest/components/icons/languages/JavaScript';
import { IconPython } from '@inngest/components/icons/languages/Python';

export const languages = ['elixir', 'go', 'js', 'py'] as const;
type Language = (typeof languages)[number];
function isLanguage(language: string): language is Language {
  return languages.includes(language as Language);
}

const languageInfo = {
  elixir: {
    Icon: IconElixir,
    text: 'Elixir',
  },
  go: {
    Icon: IconGo,
    text: 'Go',
  },
  js: {
    Icon: IconJavaScript,
    text: 'JavaScript',
  },
  py: {
    Icon: IconPython,
    text: 'Python',
  },
} as const satisfies { [key in Language]: { Icon: React.ComponentType; text: string } };

type Props = {
  language: string | null | undefined;
};

export function LanguageInfo({ language }: Props) {
  if (!language) {
    return '-';
  }

  let Icon = null;
  let text = language;
  if (isLanguage(language)) {
    const info = languageInfo[language];
    Icon = info.Icon;
    text = info.text;
  }

  return (
    <span className="flex items-center">
      {Icon && <Icon className="mr-1 shrink-0 text-slate-500" size={20} />}
      <span className="truncate">{text}</span>
    </span>
  );
}
