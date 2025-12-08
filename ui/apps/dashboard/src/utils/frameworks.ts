const frameworks = ['express', 'nextjs'] as const;
export type Framework = (typeof frameworks)[number];

export function isFramework(value: unknown): value is Framework {
  return frameworks.includes(value as Framework);
}
