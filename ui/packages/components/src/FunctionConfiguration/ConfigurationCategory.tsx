import { Children } from 'react';

type ConfigurationCategoryProps = {
  title?: string;
  children: React.ReactNode;
};

export default function ConfigurationCategory({ title, children }: ConfigurationCategoryProps) {
  if (Children.toArray(children).length == 0) {
    return null;
  }

  return (
    <div className="inline-flex flex-col items-start justify-start px-4 pb-6 pt-4">
      <h2 className="text-light pb-3 text-xs uppercase leading-4 tracking-wider">{title}</h2>
      <div className="flex flex-col space-y-6 self-stretch">{children}</div>
    </div>
  );
}
