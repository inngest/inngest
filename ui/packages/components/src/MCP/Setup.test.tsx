import type { ReactNode } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { MCPSetup } from './Setup';

vi.mock('@tanstack/react-query', () => ({
  useQuery: () => ({ isPending: true, refetch: vi.fn() }),
}));

vi.mock('@tanstack/react-router', () => ({
  ClientOnly: ({ children }: { children: ReactNode }) => children,
  Link: ({ children, to }: { children: ReactNode; to: string }) => <a href={to}>{children}</a>,
}));

vi.mock('@inngest/components/CodeBlock/CommandBlock', () => {
  const Container = ({ children }: { children: ReactNode }) => <div>{children}</div>;
  return {
    default: Object.assign(
      ({ currentTabContent }: { currentTabContent: { content: string } }) => (
        <pre>{currentTabContent.content}</pre>
      ),
      { Wrapper: Container, Header: Container, Tabs: () => null, CopyButton: () => null }
    ),
  };
});

afterEach(cleanup);

const cloudProps = {
  apiKeysHref: '/settings/api-keys',
  bearerTokenEnvVar: 'INNGEST_API_KEY',
  endpoint: 'https://api.inngest.com/mcp',
  operationsEndpoint: 'https://api.inngest.com/v2/operations',
};

describe('MCP setup', () => {
  it('defaults Cloud setup to OAuth without API-key headers', () => {
    render(<MCPSetup {...cloudProps} />);
    expect(screen.getByText('Sign in and approve access')).toBeTruthy();
    expect(
      screen.getByText('claude mcp add --transport http inngest-cloud https://api.inngest.com/mcp')
    ).toBeTruthy();
    expect(screen.queryByText('Create an API key')).toBeNull();
    expect(screen.getByRole('tab', { name: 'Cursor' })).toBeTruthy();

    fireEvent.mouseDown(screen.getByRole('tab', { name: 'Codex CLI' }), {
      button: 0,
      ctrlKey: false,
    });
    expect(
      screen.getByText('codex mcp add inngest-cloud --url https://api.inngest.com/mcp')
    ).toBeTruthy();
    expect(screen.getByText('codex mcp login inngest-cloud')).toBeTruthy();

    fireEvent.mouseDown(screen.getByRole('tab', { name: 'Cursor' }), { button: 0, ctrlKey: false });
    expect(screen.getByText(/Until Cursor supports Client ID Metadata Documents/)).toBeTruthy();
    expect(JSON.parse(screen.getByText(/"CLIENT_ID"/).textContent ?? '')).toEqual({
      mcpServers: {
        'inngest-cloud': {
          url: cloudProps.endpoint,
          auth: { CLIENT_ID: 'inngest-mcp-static' },
        },
      },
    });
    expect(screen.queryByText(/"Authorization": "Bearer/)).toBeNull();
  });

  it('keeps API-key setup available and can switch back to OAuth', () => {
    render(<MCPSetup {...cloudProps} />);
    fireEvent.click(screen.getByRole('button', { name: 'View API-key instructions' }));
    expect(screen.getByText('Create an API key')).toBeTruthy();
    expect(screen.getByText(/claude mcp add.*--header/)).toBeTruthy();
    expect(screen.getByRole('tab', { name: 'Cursor' })).toBeTruthy();
    expect(screen.queryByText('Sign in and approve access')).toBeNull();

    fireEvent.mouseDown(screen.getByRole('tab', { name: 'Cursor' }), { button: 0, ctrlKey: false });
    expect(screen.getByText(/"Authorization": "Bearer/)).toBeTruthy();
    expect(screen.queryByText(/"CLIENT_ID"/)).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'View OAuth instructions' }));
    expect(screen.getByText('Sign in and approve access')).toBeTruthy();
    expect(screen.queryByText('Create an API key')).toBeNull();
    expect(screen.getByRole('tab', { name: 'Claude Code' }).getAttribute('aria-selected')).toBe(
      'true'
    );
  });

  it('leaves Dev Server setup unauthenticated', () => {
    render(
      <MCPSetup
        endpoint="http://localhost:8288/mcp"
        operationsEndpoint="http://localhost:8288/api/v2/operations"
        isDevServer
      />
    );
    expect(
      screen.getByText('claude mcp add --transport http inngest-dev http://localhost:8288/mcp')
    ).toBeTruthy();
    expect(screen.getByRole('tab', { name: 'Cursor' })).toBeTruthy();
    expect(screen.queryByText('Sign in and approve access')).toBeNull();
    expect(screen.queryByRole('button', { name: 'View API-key instructions' })).toBeNull();
    fireEvent.mouseDown(screen.getByRole('tab', { name: 'Cursor' }), { button: 0, ctrlKey: false });
    expect(JSON.parse(screen.getByText(/"inngest-dev"/).textContent ?? '')).toEqual({
      mcpServers: { 'inngest-dev': { url: 'http://localhost:8288/mcp' } },
    });
  });
});
