import type { ButtonAppearance, ButtonKind, ButtonSize } from './Button';

interface ButtonColorParams {
  kind: ButtonKind;
  appearance: ButtonAppearance;
  loading?: boolean;
}

interface ButtonSizeParams {
  size: ButtonSize;
}

interface ButtonSizeStyleParams extends ButtonSizeParams {
  icon?: React.ReactNode;
  label?: React.ReactNode;
}

export const getButtonColors = ({ kind, appearance, loading }: ButtonColorParams) => {
  const solidButtonStyles = {
    primary: loading
      ? 'bg-primary-xSubtle text-onContrast'
      : 'bg-primary-intense focus:bg-primary-xIntense hover:bg-primary-xIntense active:bg-primary-2xIntense disabled:bg-primary-xSubtle text-onContrast',
    secondary: '', // NOOP: there are no designs for secondary solid buttons,
    danger: loading
      ? 'bg-tertiary-xSubtle text-onContrast'
      : 'bg-tertiary-intense focus:bg-primary-xIntense hover:bg-tertiary-xIntense active:bg-tertiary-2xIntense disabled:bg-tertiary-xSubtle text-onContrast',
  };

  const outlinedButtonStyles = {
    primary: loading
      ? 'border border-subtle text-primary-moderate'
      : 'border border-muted text-primary-intense focus:bg-subtle hover:bg-subtle active:bg-muted disabled:bg-disabled disabled:text-primary-xSubtle',
    secondary: loading
      ? 'border border-subtle text-foreground-subtle'
      : 'border border-muted text-foreground-base focus:bg-subtle hover:bg-subtle active:bg-muted disabled:bg-disabled disabled:text-foreground-disabled',
    danger: loading
      ? 'border border-subtle text-tertiary-moderate'
      : 'border border-muted text-tertiary-intense focus:bg-subtle hover:bg-subtle active:bg-muted disabled:bg-disabled disabled:text-tertiary-xSubtle',
  };

  const ghostButtonStyles = {
    primary: loading
      ? 'text-primary-moderate'
      : 'text-primary-intense focus:bg-subtle hover:bg-subtle active:bg-muted disabled:text-primary-xSubtle',
    secondary: loading
      ? 'text-foreground-subtle'
      : 'text-foreground-base focus:bg-subtle hover:bg-subtle active:bg-muted disabled:text-foreground-disabled',
    danger: loading
      ? 'text-tertiary-moderate'
      : 'text-tertiary-intense focus:bg-subtle hover:bg-subtle active:bg-muted disabled:bg-disabled disabled:text-tertiary-xSubtle',
  };

  if (appearance === 'solid') {
    return solidButtonStyles[kind];
  } else if (appearance === 'outlined') {
    return outlinedButtonStyles[kind];
  } else {
    return ghostButtonStyles[kind];
  }
};

export const getKeyColor = ({ appearance, kind }: ButtonColorParams) => {
  const defaultKeyStyles = {
    primary: 'bg-indigo-500/80',
    secondary: 'bg-emerald-600/80',
    danger: 'bg-rose-700/80',
  };
  if (appearance === 'solid' && kind === 'primary') {
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
    small: 'h-6 p-1.5',
    medium: 'h-8 p-1.5',
    large: 'h-10 p-1.5',
  };

  const sizeStyles = {
    small: 'h-6 text-xs leading-[18px] px-3 py-1.5',
    medium: 'h-8 text-xs leading-[18px] px-3 py-1.5',
    large: 'h-10 text-xs leading-[18px] px-3 py-1.5',
  };

  return icon && !label ? iconOnlySizeStyles[size] : sizeStyles[size];
};

export const getDisabledStyles = ({ appearance }: ButtonColorParams) => {
  if (appearance === 'solid') {
    return 'disabled:cursor-not-allowed disabled:text-slate-400 disabled:bg-slate-200 dark:disabled:text-slate-500 dark:disabled:bg-slate-800 ';
  } else if (appearance === 'outlined') {
    return 'disabled:cursor-not-allowed disabled:text-slate-400 disabled:border-slate-200 disabled:bg-slate-100 dark:disabled:text-slate-500 dark:disabled:border-slate-800 dark:disabled:bg-slate-900';
  }
  return 'disabled:cursor-not-allowed disabled:text-slate-400 dark:disabled:text-slate-500 disabled:hover:no-underline';
};

export const getIconSizeStyles = ({ size }: ButtonSizeParams) => {
  const sizeStyles = {
    small: 'h-4 w-4',
    medium: 'h-4 w-4',
    large: 'h-4 w-4',
  };

  return sizeStyles[size];
};

export const getSpinnerStyles = ({ appearance, kind }: ButtonColorParams) => {
  const defaultSpinnerStyles = {
    primary: 'fill-primary-moderate',
    secondary: 'fill-foreground-subtle',
    danger: 'fill-tertiary-moderate',
  };
  if (appearance === 'outlined') {
    return defaultSpinnerStyles[kind];
  }
  return 'fill-foreground-onContrast';
};
