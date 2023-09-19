const kindColors = {
  default: 'slate-800',
  primary: 'indigo-500',
  success: 'emerald-600',
  danger: 'rose-700',
};

interface ButtonColorParams {
  kind: 'default' | 'primary' | 'success' | 'danger';
  appearance: 'solid' | 'outlined' | 'text';
}

interface ButtonSizeParams {
  size: 'small' | 'regular' | 'large';
}

interface ButtonSizeStyleParams extends ButtonSizeParams {
  icon?: React.ReactNode;
  label?: React.ReactNode;
}

export const getButtonColors = ({ kind, appearance }: ButtonColorParams) => {
  const textColors = {
    default: 'slate-500',
    primary: 'indigo-500',
    success: 'emerald-600',
    danger: 'rose-500',
  };

  if (appearance === 'solid') {
    return `bg-${kindColors[kind]} border-t border-white/10 hover:bg-${kindColors[kind]}/80 text-slate-100 hover:text-white`;
  } else if (appearance === 'outlined') {
    return `bg-${kindColors[kind]}/20 border border-${kindColors[kind]}/80 hover:border-${kindColors[kind]} text-slate-200 hover:text-white`;
  } else {
    return `text-${textColors[kind]} hover:text-${textColors[kind]}/80`;
  }
};

export const getKeyColor = ({ appearance, kind }: ButtonColorParams) => {
  if (appearance === 'solid' && kind === 'default') {
    return `bg-slate-800`;
  } else if (appearance === 'solid') {
    return `bg-slate-800/20`;
  }
  return `bg-${kindColors[kind]}/80`;
};

export const getButtonSizeStyles = ({ size, icon, label }: ButtonSizeStyleParams) => {
  const iconOnlySizeStyles = {
    small: 'w-7 h-7',
    regular: 'w-8 h-8',
    large: 'w-10 h-10',
  };

  const sizeStyles = {
    small: 'text-xs px-2.5 h-7 leading-7',
    regular: 'text-sm px-2.5 h-8 leading-8',
    large: 'text-base px-2.5 h-10 leading-10',
  };

  return icon && !label ? iconOnlySizeStyles[size] : sizeStyles[size];
};

export const getDisabledStyles = () => {
  return 'disabled:text-slate-500 disabled:cursor-not-allowed disabled:bg-slate-800 disabled:hover:bg-slate-800 disabled:border-slate-800';
};

export const getIconSizeStyles = ({ size }: ButtonSizeParams) => {
  const sizeStyles = {
    small: 'icon-sm',
    regular: 'icon-base',
    large: 'icon-lg',
  };

  return sizeStyles[size];
};
