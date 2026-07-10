import type { ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { AITrace } from './AITrace';

vi.mock('../DetailsCard/Element', () => ({
  ElementWrapper: ({ label, children }: { label: string; children: ReactNode }) => (
    <div>
      <span>{label}</span>
      <div>{children}</div>
    </div>
  ),
  TextElement: ({ children }: { children: ReactNode }) => <span>{children}</span>,
}));

describe('AITrace', () => {
  it('renders canonical AI metadata labels', () => {
    render(
      <AITrace
        aiInfo={{
          model: 'gpt-4o-mini-2024-07-18',
          inputTokens: 16,
          outputTokens: 19,
          totalTokens: 35,
        }}
      />
    );

    expect(screen.getByText('Model')).toBeTruthy();
    expect(screen.getByText('Input Tokens')).toBeTruthy();
    expect(screen.getByText('Output Tokens')).toBeTruthy();
    expect(screen.getByText('Total Tokens')).toBeTruthy();
    expect(screen.queryByText('Prompt Tokens')).toBeNull();
    expect(screen.queryByText('Completion Tokens')).toBeNull();
  });
});
