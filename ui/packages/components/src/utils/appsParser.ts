const languages = ['elixir', 'go', 'js', 'py', 'typescript', 'rust'] as const;

type Language = (typeof languages)[number];

function isLanguage(language: string): language is Language {
  return languages.includes(language as Language);
}

const languageMap: Record<Language, string> = {
  elixir: 'Elixir',
  go: 'Go',
  js: 'JavaScript',
  py: 'Python',
  typescript: 'TypeScript',
  rust: 'Rust',
};

export function transformLanguage(language?: string): string | undefined {
  if (!language) return undefined;
  return isLanguage(language) ? languageMap[language] : language;
}

const frameworks = [
  'aws-lambda',
  'deno/fresh',
  'django',
  'express',
  'fast_api',
  'fastify',
  'flask',
  'h3',
  'koa',
  'nextjs',
  'nuxt',
  'phoenix',
  'redwoodjs',
  'remix',
  'sveltekit',
] as const;

type Framework = (typeof frameworks)[number];

function isFramework(framework: string): framework is Framework {
  return frameworks.includes(framework as Framework);
}

const frameworkMap: Record<string, string> = {
  'aws-lambda': 'AWS Lambda',
  'deno/fresh': 'Fresh',
  django: 'Django',
  express: 'Express',
  fast_api: 'FastAPI',
  fastify: 'Fastify',
  flask: 'Flask',
  h3: 'h3',
  koa: 'Koa',
  nextjs: 'Next.js',
  nuxt: 'Nuxt',
  phoenix: 'Phoenix',
  redwoodjs: 'RedwoodJS',
  remix: 'Remix',
  sveltekit: 'SvelteKit',
};

export function transformFramework(framework?: string | null): string | undefined {
  if (!framework) return undefined;
  return isFramework(framework) ? frameworkMap[framework] : framework;
}

export const platforms = ['cloudflare-pages', 'railway', 'render', 'vercel'] as const;

type Platform = (typeof platforms)[number];

function isPlatform(platform: string): platform is Platform {
  return platforms.includes(platform as Platform);
}

export const platformMap: Record<string, string> = {
  'cloudflare-pages': 'Cloudflare Pages',
  railway: 'Railway',
  render: 'Render',
  vercel: 'Vercel',
};

export function transformPlatform(platform?: string | null): string | undefined {
  if (!platform) return undefined;
  return isPlatform(platform) ? platformMap[platform] : platform;
}
