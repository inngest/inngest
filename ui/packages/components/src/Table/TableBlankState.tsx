import React from 'react';

type TableBlankStateProps = {
  icon?: React.ReactNode;
  actions: React.ReactNode;
  title?: React.ReactNode;
  description?: React.ReactNode;
};

export function TableBlankState({ actions, title, description, icon }: TableBlankStateProps) {
  const iconElement = React.isValidElement(icon)
    ? React.cloneElement(icon as React.ReactElement, {
        className: 'h-7 w-7',
      })
    : null;

  return (
    <div className="text-basis mt-36 flex flex-col items-center justify-center gap-5">
      {iconElement && (
        <div className="bg-canvasSubtle text-light rounded-md p-3 ">{iconElement}</div>
      )}
      <div className="text-center">
        <p className="mb-1.5 text-xl">{title}</p>
        <p className="text-subtle max-w-md text-sm">{description}</p>
      </div>
      <div className="flex items-center gap-3">{actions}</div>
    </div>
  );
}
