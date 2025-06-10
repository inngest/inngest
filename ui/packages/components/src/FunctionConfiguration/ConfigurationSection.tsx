import { Children } from 'react';

type ConfigurationSectionProps = {
  title?: string;
  children: React.ReactNode;
};

export default function ConfigurationSection({ title, children }: ConfigurationSectionProps) {
  if (Children.count(children) == 0) {
    return <></>;
  }

  return (
    <div>
      {/* TODO do we want font weight 450 specifically? */}
      <h3 className="text-basis mb-1 text-sm font-medium">{title}</h3>
      {children}
    </div>
  );
}
