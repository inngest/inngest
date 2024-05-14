//
// This will futureproof us a bit if we switch to manual or system color scheme
export const colorScheme = (): 'light' | 'dark' =>
  localStorage.theme === 'dark' ||
  window.matchMedia('(prefers-color-scheme: dark)').matches ||
  document.documentElement.classList.contains('dark')
    ? 'dark'
    : 'light';

export const isDark = () => colorScheme() === 'dark';
