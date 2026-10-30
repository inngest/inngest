import { cleanup, fireEvent, render } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import CommandBlock from '@inngest/components/CodeBlock/CommandBlock';

const fixture = vi.hoisted(() => ({
  monaco: {
    editor: { defineTheme: vi.fn() },
    languages: {
      register: vi.fn(),
      setMonarchTokensProvider: vi.fn(),
      typescript: { typescriptDefaults: { setDiagnosticsOptions: vi.fn() } },
    },
  },
  colors: vi.fn(() => ({ 'editor.background': '#121212' })),
  rules: vi.fn(() => [{ token: 'identifier', foreground: '#89c0ea' }]),
}));
vi.mock('@monaco-editor/react', () => ({
  useMonaco: () => fixture.monaco,
  default: ({ value }: { value: string }) => <pre>{value}</pre>,
}));
vi.mock('@inngest/components/utils/monaco', () => ({
  FONT: { font: 'monospace', size: 13 },
  LINE_HEIGHT: 26,
  shellLanguageTokens: {},
  createColors: fixture.colors,
  createRules: fixture.rules,
}));
beforeEach(() => {
  vi.clearAllMocks();
  document.documentElement.classList.add('dark');
  document.documentElement.style.removeProperty(
    '--color-background-codeEditor',
  );
});
afterEach(() => {
  cleanup();
  document.documentElement.classList.remove('dark');
  document.documentElement.style.removeProperty(
    '--color-background-codeEditor',
  );
});
it('uses readable fallback colors until CSS tokens load, then applies the shared theme', () => {
  const view = render(
    <CommandBlock
      currentTabContent={{
        title: 'npm',
        content: 'npm install inngest',
        language: 'shell',
      }}
    />,
  );
  expect(fixture.monaco.editor.defineTheme).toHaveBeenLastCalledWith(
    'inngest-theme',
    {
      base: 'vs-dark',
      inherit: true,
      rules: [],
      colors: {},
    },
  );
  expect(fixture.colors).not.toHaveBeenCalled();
  document.documentElement.style.setProperty(
    '--color-background-codeEditor',
    '18 18 18',
  );
  const stylesheet = document.createElement('link');
  stylesheet.rel = 'stylesheet';
  document.head.appendChild(stylesheet);
  fireEvent.load(stylesheet);
  expect(fixture.monaco.editor.defineTheme).toHaveBeenLastCalledWith(
    'inngest-theme',
    {
      base: 'vs-dark',
      inherit: true,
      rules: [{ token: 'identifier', foreground: '#89c0ea' }],
      colors: { 'editor.background': '#121212' },
    },
  );
  view.unmount();
  const count = fixture.monaco.editor.defineTheme.mock.calls.length;
  fireEvent.load(stylesheet);
  expect(fixture.monaco.editor.defineTheme).toHaveBeenCalledTimes(count);
  stylesheet.remove();
});
