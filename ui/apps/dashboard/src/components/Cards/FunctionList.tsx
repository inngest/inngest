import { FunctionListItem } from './FunctionListItem';

interface Props {
  functions: { name: string; slug: string }[];
  baseHref: string;
  status: 'active' | 'removed';
}

export const ulClassname = 'divide-y divide-slate-800 bg-slate-900 overflow-hidden rounded-lg';

export function FunctionList({ functions, baseHref, status }: Props) {
  let title = 'Deployed Functions';
  if (status === 'removed') {
    title = 'Removed Functions';
  }

  let content: React.ReactNode | undefined = undefined;
  if (functions.length > 0) {
    content = (
      <ul className={ulClassname}>
        {functions.map((fn) => {
          return (
            <FunctionListItem
              key={fn.name}
              name={fn.name}
              href={`${baseHref}/${encodeURIComponent(fn.slug)}`}
              status={status}
            />
          );
        })}
      </ul>
    );
  }

  return (
    <div>
      <h4 className="px-4 py-3 text-base font-medium text-slate-700">{title}</h4>
      {content}
    </div>
  );
}
