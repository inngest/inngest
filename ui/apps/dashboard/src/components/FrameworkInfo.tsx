import { IconAWSLambda } from '@inngest/components/icons/frameworks/AWSLambda';
import { IconDjango } from '@inngest/components/icons/frameworks/Django';
import { IconExpress } from '@inngest/components/icons/frameworks/Express';
import { IconFastAPI } from '@inngest/components/icons/frameworks/FastAPI';
import { IconFastify } from '@inngest/components/icons/frameworks/Fastify';
import { IconFlask } from '@inngest/components/icons/frameworks/Flask';
import { IconFresh } from '@inngest/components/icons/frameworks/Fresh';
import { IconH3 } from '@inngest/components/icons/frameworks/H3';
import { IconNext } from '@inngest/components/icons/frameworks/Next';
import { IconNuxt } from '@inngest/components/icons/frameworks/Nuxt';
import { IconPhoenix } from '@inngest/components/icons/frameworks/Phoenix';
import { IconRedwood } from '@inngest/components/icons/frameworks/Redwood';
import { IconRemix } from '@inngest/components/icons/frameworks/Remix';
import { IconSvelte } from '@inngest/components/icons/frameworks/Svelte';

export const frameworks = [
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

const frameworkInfo = {
  'aws-lambda': {
    Icon: IconAWSLambda,
    text: 'AWS Lambda',
  },
  'deno/fresh': {
    Icon: IconFresh,
    text: 'Fresh',
  },
  django: {
    Icon: IconDjango,
    text: 'Django',
  },
  express: {
    Icon: IconExpress,
    text: 'Express',
  },
  fast_api: {
    Icon: IconFastAPI,
    text: 'FastAPI',
  },
  fastify: {
    Icon: IconFastify,
    text: 'Fastify',
  },
  flask: {
    Icon: IconFlask,
    text: 'Flask',
  },
  h3: {
    Icon: IconH3,
    text: 'h3',
  },
  koa: {
    // Couldn't find a logo that wasn't just the name
    Icon: null,
    text: 'Koa',
  },
  nextjs: {
    Icon: IconNext,
    text: 'Next.js',
  },
  nuxt: {
    Icon: IconNuxt,
    text: 'Nuxt',
  },
  phoenix: {
    Icon: IconPhoenix,
    text: 'Phoenix',
  },
  redwoodjs: {
    Icon: IconRedwood,
    text: 'RedwoodJS',
  },
  remix: {
    Icon: IconRemix,
    text: 'Remix',
  },
  sveltekit: {
    Icon: IconSvelte,
    text: 'Svelte',
  },
} as const satisfies { [key in Framework]: { Icon: React.ComponentType | null; text: string } };

type Props = {
  framework: string | null | undefined;
};

export function FrameworkInfo({ framework }: Props) {
  if (!framework) {
    return '-';
  }

  let Icon = null;
  let text = framework;
  if (isFramework(framework)) {
    const info = frameworkInfo[framework];
    Icon = info.Icon;
    text = info.text;
  }

  return (
    <span className="flex items-center">
      {Icon && <Icon className="mr-1 shrink-0 text-slate-500" size={20} />}
      <span className="truncate">{text}</span>
    </span>
  );
}
