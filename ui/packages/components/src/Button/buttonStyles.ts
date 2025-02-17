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
      ? 'bg-btnPrimaryDisabled text-alwaysWhite'
      : 'bg-btnPrimary focus:bg-btnPrimaryPressed hover:bg-btnPrimaryHover active:bg-btnPrimaryPressed disabled:bg-btnPrimaryDisabled text-alwaysWhite',
    secondary: '', // NOOP: there are no designs for secondary solid buttons,
    danger: loading
      ? 'bg-btnDangerDisabled text-alwaysWhite'
      : 'bg-btnDanger focus:bg-btnDangerPressed hover:bg-btnDangerHover active:bg-btnDangerPressed disabled:bg-btnDangerDisabled text-alwaysWhite',
  };

  const outlinedButtonStyles = {
    primary: loading
      ? 'border border-subtle text-btnPrimaryDisabled bg-canvasBase'
      : 'border border-muted text-btnPrimary bg-canvasBase focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted disabled:border-disabled disabled:bg-disabled disabled:text-btnPrimaryDisabled',
    secondary: loading
      ? 'border border-subtle text-foreground-subtle bg-canvasBase'
      : 'border border-muted text-basis bg-canvasBase focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted disabled:border-disabled disabled:bg-disabled disabled:text-disabled',
    danger: loading
      ? 'border border-subtle text-btnDangerDisabled bg-canvasBase'
      : 'border border-muted text-btnDanger bg-canvasBase focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted disabled:border-disabled disabled:bg-disabled disabled:text-btnDangerDisabled',
  };

  const ghostButtonStyles = {
    primary: loading
      ? 'text-btnPrimaryDisabled'
      : 'text-btnPrimary focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted disabled:bg-disabled disabled:text-btnPrimaryDisabled',
    secondary: loading
      ? 'text-foreground-subtle'
      : 'text-basis focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted disabled:bg-disabled disabled:text-disabled',
    danger: loading
      ? 'text-btnDangerDisabled'
      : 'text-btnDanger focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted disabled:bg-disabled disabled:text-btnDangerDisabled',
  };

  if (appearance === 'solid') {
    return solidButtonStyles[kind];
  } else if (appearance === 'outlined') {
    return outlinedButtonStyles[kind];
  } else {
    return ghostButtonStyles[kind];
  }
};

export const getButtonSizeStyles = ({ size, icon, label }: ButtonSizeStyleParams) => {
  const iconOnlySizeStyles = {
    small: 'h-6 p-1.5',
    medium: 'h-8 p-1.5',
    large: 'h-10 p-1.5',
  };

  const sizeStyles = {
    small: 'h-6 text-xs leading-[18px] px-2 py-1.5',
    medium: 'h-8 text-xs leading-[18px] px-3 py-1.5',
    large: 'h-10 text-xs leading-[18px] px-3 py-1.5',
  };

  return icon && !label ? iconOnlySizeStyles[size] : sizeStyles[size];
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
    primary: 'fill-btnPrimary',
    secondary: 'fill-subtle',
    danger: 'fill-btnDanger',
  };
  if (appearance === 'outlined' || appearance === 'ghost') {
    return defaultSpinnerStyles[kind];
  }
  return 'fill-alwaysWhite';
};
