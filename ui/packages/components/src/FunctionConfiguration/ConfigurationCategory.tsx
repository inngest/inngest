import { Children } from 'react';

type ConfigurationCategoryProps = {
  title?: string;
  children: React.ReactNode;
};

export function ConfigurationCategory({ title, children }: ConfigurationCategoryProps) {
  if (Children.count(children) == 0) {
    return <></>;
  }

  return (
    <div className="inline-flex flex-col items-start justify-start px-4 pb-6 pt-4">
      {/* TODO letter spacing */}
      <h2 className="text-light pb-3 text-xs font-medium uppercase leading-4 tracking-wider">
        {title}
      </h2>
      {children}
    </div>
  );
}
