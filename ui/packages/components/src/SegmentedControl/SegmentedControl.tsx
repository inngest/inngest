import { Children, cloneElement, forwardRef, isValidElement, useState } from 'react';

import { cn } from '../utils/classNames';

type SegmentedProps = {
  defaultValue?: string;
};

export default function SegmentedControl({
  children,
  defaultValue,
}: React.PropsWithChildren<SegmentedProps>) {
  const [activeValue, setActiveValue] = useState(defaultValue);

  const mappedChildren = Children.map(children, (child) => {
    if (isValidElement<SegmentedButtonProps>(child) && child.type === SegmentedControl.Button) {
      return cloneElement(child, {
        isActive: child.props.value === activeValue,
        onClick: () => {
          setActiveValue(child.props.value);
          child.props.onClick?.();
        },
      });
    }
    return child;
  });

  return <ul className="bg-canvasMuted flex rounded-full">{mappedChildren}</ul>;
}

type SegmentedButtonProps = {
  value: string;
  isActive?: boolean;
  onClick?: () => void;
  icon?: React.ReactNode;
  iconSide?: 'right' | 'left';
};

const Button = forwardRef<HTMLButtonElement, React.PropsWithChildren<SegmentedButtonProps>>(
  ({ children, value, isActive, icon, iconSide, onClick, ...props }, ref) => {
    const iconElement = isValidElement(icon)
      ? cloneElement(icon as React.ReactElement, {
          className: cn('h-4 w-4 ', children ? 'mx-0' : 'mx-[5px]', icon.props.className),
        })
      : null;

    return (
      <li className="h-7" value={value}>
        <button
          title={children ? undefined : value}
          className={cn(
            isActive ? 'text-basis bg-canvasBase border-muted' : ' text-muted border-transparent',
            children ? 'flex items-center gap-1 px-3' : '',
            'hover:bg-canvasSubtle hover:text-basis disabled:text-disabled h-full rounded-full border text-sm disabled:cursor-not-allowed'
          )}
          onClick={onClick}
          ref={ref}
          {...props}
        >
          {icon && iconSide === 'left' && <span className="">{iconElement}</span>}
          {icon && !iconSide && <span>{iconElement}</span>}
          {children}
          {icon && iconSide === 'right' && <span className="">{iconElement}</span>}
        </button>
      </li>
    );
  }
);

SegmentedControl.Button = Button;
