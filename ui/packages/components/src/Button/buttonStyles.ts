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
  const solidButtonStyles = {
    default:
      'bg-slate-800 border-t border-white/10 hover:bg-slate-800/80 text-slate-100 hover:text-white',
    primary:
      'bg-indigo-500 border-t border-white/10 hover:bg-indigo-500/80 text-slate-100 hover:text-white',
    success:
      'bg-emerald-600 border-t border-white/10 hover:bg-emerald-600/80 text-slate-100 hover:text-white',
    danger:
      'bg-rose-700 border-t border-white/10 hover:bg-rose-700/80 text-slate-100 hover:text-white',
  };

  const outlinedButtonStyles = {
    default:
      'bg-white dark:bg-slate-800/20 border dark:border-slate-800/80 hover:dark:border-slate-800 text-slate-700 dark:text-slate-200 hover:bg-slate-100 hover:text-indigo-500 dark:hover:text-white',
    primary:
      'bg-white dark:bg-indigo-500/20 border border-indigo-500/80 hover:border-indigo-500 text-indigo-500 dark:text-slate-200 hover:bg-slate-100 dark:hover:text-white',
    success:
      'bg-white dark:bg-emerald-600/20 border border-emerald-600/80 hover:border-emerald-600 text-emerald-600 dark:text-slate-200 hover:bg-slate-100 dark:hover:text-white',
    danger:
      'bg-white dark:bg-rose-700/20 border border-rose-700/80 hover:border-rose-700 text-rose-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:text-white',
  };

  const textButtonStyles = {
    default: 'text-slate-500 hover:text-slate-500/80 hover:underline',
    primary: 'text-indigo-500 hover:text-indigo-500/80 hover:underline',
    success: 'text-emerald-600 hover:text-emerald-600/80 hover:underline',
    danger: 'text-rose-500 hover:text-rose-500/80 hover:underline',
  };

  if (appearance === 'solid') {
    return solidButtonStyles[kind];
  } else if (appearance === 'outlined') {
    return outlinedButtonStyles[kind];
  } else {
    return textButtonStyles[kind];
  }
};

export const getKeyColor = ({ appearance, kind }: ButtonColorParams) => {
  const defaultKeyStyles = {
    default: 'bg-slate-800/80',
    primary: 'bg-indigo-500/80',
    success: 'bg-emerald-600/80',
    danger: 'bg-rose-700/80',
  };
  if (appearance === 'solid' && kind === 'default') {
    return 'bg-slate-900';
  } else if (appearance === 'solid') {
    return 'bg-slate-800/20';
  } else if (appearance === 'outlined') {
    return `text-white ${defaultKeyStyles[kind]}`;
  }
  return defaultKeyStyles[kind];
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

export const getDisabledStyles = ({ appearance, kind }: ButtonColorParams) => {
  if (appearance === 'solid') {
    return 'disabled:cursor-not-allowed disabled:text-slate-400 disabled:bg-slate-200 dark:disabled:text-slate-500 dark:disabled:bg-slate-800 ';
  } else if (appearance === 'outlined') {
    return 'disabled:cursor-not-allowed disabled:text-slate-400 disabled:border-slate-200 disabled:bg-slate-100 dark:disabled:text-slate-500 dark:disabled:border-slate-800 dark:disabled:bg-slate-900';
  }
  return 'disabled:cursor-not-allowed disabled:text-slate-400 dark:disabled:text-slate-500 disabled:hover:no-underline';
};

export const getIconSizeStyles = ({ size }: ButtonSizeParams) => {
  const sizeStyles = {
    small: 'h-3.5 w-3.5',
    regular: 'h-4 w-4',
    large: 'h-5 w-5',
  };

  return sizeStyles[size];
};

export const getSpinnerStyles = ({ appearance, kind }: ButtonColorParams) => {
  const defaultSpinnerStyles = {
    default: 'fill-slate-700 dark:fill-white',
    primary: 'fill-indigo-500 dark:fill-white',
    success: 'fill-emerald-600 dark:fill-white',
    danger: 'fill-rose-700 dark:fill-white',
  };
  if (appearance === 'outlined') {
    return defaultSpinnerStyles[kind];
  }
  return 'fill-white';
};
