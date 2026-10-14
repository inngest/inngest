// Visual primitives shared by Radix action menus and Headless UI listboxes.
// Keep positioning and interaction behavior in the consuming component.
export const menuPanelClassName =
  'dropdown-panel border-subtle bg-canvasBase shadow-primary rounded-md border p-1 text-[13px] leading-5 outline-none';

export const menuItemClassName =
  'dropdown-item text-basis flex min-h-8 select-none items-center gap-2 rounded px-2 py-1.5 text-[13px] font-normal leading-5 outline-none';

export const menuDangerItemClassName =
  'text-error hover:bg-error data-[highlighted]:bg-error data-[disabled]:hover:bg-transparent [&>svg]:text-error data-[disabled]:[&>svg]:text-disabled';

export const headlessMenuItemClassName = `${menuItemClassName} hover:bg-canvasSubtle ui-active:bg-canvasSubtle ui-selected:bg-canvasSubtle ui-disabled:text-disabled ui-disabled:cursor-not-allowed ui-disabled:hover:bg-transparent`;

export const menuSeparatorClassName = 'border-subtle -mx-1 my-1 border-t';

export const menuLabelClassName = 'text-muted px-2 pb-1 pt-2 text-xs leading-5';

export const menuSearchClassName = 'px-2 py-2';

export const menuFooterClassName =
  'border-subtle -mx-1 mt-1 flex min-h-10 items-center gap-2 border-t px-3 py-1';
