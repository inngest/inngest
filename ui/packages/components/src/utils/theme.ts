//
// This will futureproof us a bit if we switch to manual or system color scheme.
// If an element ref is provided we look for a dark class in any parents.
// This is useful for components that might have dark theme sections inside a general light theme
export const colorScheme = (elementRef?: HTMLElement): 'light' | 'dark' =>
  localStorage.theme === 'dark' ||
  window.matchMedia('(prefers-color-scheme: dark)').matches ||
  (elementRef ? elementRef.closest('.dark') : document.documentElement.classList.contains('dark'))
    ? 'dark'
    : 'light';

export const isDark = (elementRef?: HTMLElement) => colorScheme(elementRef) === 'dark';
