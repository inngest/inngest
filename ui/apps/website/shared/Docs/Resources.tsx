import Link from 'next/link';
import {
  ArrowsPointingOutIcon,
  ChevronDoubleRightIcon,
  ClockIcon,
  PaperAirplaneIcon,
  RectangleGroupIcon,
  Square3Stack3DIcon,
} from '@heroicons/react/24/outline';
import { motion, useMotionTemplate, useMotionValue } from 'framer-motion';
import ParallelIcon from 'src/shared/Icons/Parallel';

import { GridPattern } from './GridPattern';
import { Heading } from './Heading';
import { ChatBubbleIcon } from './icons/ChatBubbleIcon';
import { EnvelopeIcon } from './icons/EnvelopeIcon';
import { UserIcon } from './icons/UserIcon';
import { UsersIcon } from './icons/UsersIcon';

// Icons available for direct use in MDX
const icons = {
  'paper-airplane': PaperAirplaneIcon,
  clock: ClockIcon,
  'arrows-pointing-out': ArrowsPointingOutIcon,
  'chevron-double-right': ChevronDoubleRightIcon,
  'square-3-stack-3d': Square3Stack3DIcon,
  parallel: ParallelIcon,
  'rectangle-group': RectangleGroupIcon,
} as const;

type IconType = keyof typeof icons;

const patterns = [
  {
    y: 16,
    squares: [
      [0, 1],
      [1, 3],
    ],
  },
  {
    y: -6,
    squares: [
      [-1, 2],
      [1, 3],
    ],
  },
  {
    y: 32,
    squares: [
      [0, 2],
      [1, 4],
    ],
  },
  {
    y: 22,
    squares: [[0, 1]],
  },
];

const resources = [
  {
    href: '/contacts',
    name: 'Contacts',
    description:
      'Learn about the contact model and how to create, retrieve, update, delete, and list contacts.',
    icon: UserIcon,
    pattern: {
      y: 16,
      squares: [
        [0, 1],
        [1, 3],
      ],
    },
  },
  {
    href: '/conversations',
    name: 'Conversations',
    description:
      'Learn about the conversation model and how to create, retrieve, update, delete, and list conversations.',
    icon: ChatBubbleIcon,
    pattern: {
      y: -6,
      squares: [
        [-1, 2],
        [1, 3],
      ],
    },
  },
  {
    href: '/messages',
    name: 'Messages',
    description:
      'Learn about the message model and how to create, retrieve, update, delete, and list messages.',
    icon: EnvelopeIcon,
    pattern: {
      y: 32,
      squares: [
        [0, 2],
        [1, 4],
      ],
    },
  },
  {
    href: '/groups',
    name: 'Groups',
    description:
      'Learn about the group model and how to create, retrieve, update, delete, and list groups.',
    icon: UsersIcon,
    pattern: {
      y: 22,
      squares: [[0, 1]],
    },
  },
];

function ResourceIcon({ icon: Icon }) {
  return (
    <div className="dark:bg-white/7.5 flex h-7 w-7 items-center justify-center rounded-full bg-slate-900/5 ring-1 ring-slate-900/25 backdrop-blur-[2px] transition duration-300 group-hover:bg-white/50 group-hover:ring-slate-900/25 dark:ring-white/15 dark:group-hover:bg-indigo-300/10 dark:group-hover:ring-indigo-400">
      <Icon className="w45 h-4 fill-slate-700/10 stroke-slate-700 transition-colors duration-300 group-hover:stroke-slate-900 dark:fill-white/10 dark:stroke-slate-400 dark:group-hover:fill-indigo-300/10 dark:group-hover:stroke-indigo-400" />
    </div>
  );
}

function ResourcePattern({ mouseX, mouseY, ...gridProps }) {
  let maskImage = useMotionTemplate`radial-gradient(180px at ${mouseX}px ${mouseY}px, white, transparent)`;
  let style = { maskImage, WebkitMaskImage: maskImage };

  return (
    <div className="pointer-events-none">
      <div className="absolute inset-0 rounded-lg transition duration-300 [mask-image:linear-gradient(white,transparent)] group-hover:opacity-50">
        <GridPattern
          width={72}
          height={56}
          x="50%"
          className="dark:fill-white/1 dark:stroke-white/2.5 absolute inset-x-0 inset-y-[-30%] h-[160%] w-full skew-y-[-18deg] fill-black/[0.02] stroke-black/5"
          {...gridProps}
        />
      </div>
      <motion.div
        className="absolute inset-0 rounded-lg bg-gradient-to-r from-indigo-50 to-sky-100 opacity-0 transition duration-300 group-hover:opacity-100 dark:from-[#202D2E] dark:to-[#303428]"
        style={style}
      />
      <motion.div
        className="absolute inset-0 rounded-lg opacity-0 mix-blend-overlay transition duration-300 group-hover:opacity-100"
        style={style}
      >
        <GridPattern
          width={72}
          height={56}
          x="50%"
          className="dark:fill-white/2.5 absolute inset-x-0 inset-y-[-30%] h-[160%] w-full skew-y-[-18deg] fill-black/50 stroke-black/70 dark:stroke-white/10"
          {...gridProps}
        />
      </motion.div>
    </div>
  );
}

export function Resource({
  resource,
}: {
  resource: {
    href: string;
    name: string;
    description: string;
    pattern: 0 | 1 | 2 | 3 | object | null;
    icon: IconType | ((any) => JSX.Element);
  };
}) {
  let mouseX = useMotionValue(0);
  let mouseY = useMotionValue(0);

  function onMouseMove({ currentTarget, clientX, clientY }) {
    let { left, top } = currentTarget.getBoundingClientRect();
    mouseX.set(clientX - left);
    mouseY.set(clientY - top);
  }
  const pattern =
    resource.pattern === null
      ? patterns[0]
      : typeof resource.pattern === 'number'
      ? patterns[resource.pattern]
      : resource.pattern;

  const icon =
    typeof resource.icon === 'string' && icons.hasOwnProperty(resource.icon)
      ? icons[resource.icon]
      : resource.icon;

  return (
    <div
      key={resource.href}
      onMouseMove={onMouseMove}
      className="dark:bg-white/2.5 group relative flex rounded-lg bg-slate-50 transition-shadow hover:shadow-md hover:shadow-slate-900/5 dark:hover:shadow-black/5"
    >
      <ResourcePattern {...pattern} mouseX={mouseX} mouseY={mouseY} />
      <div className="ring-slate-900/7.5 absolute inset-0 rounded-lg ring-1 ring-inset group-hover:ring-slate-900/10 dark:ring-white/10 dark:group-hover:ring-white/20" />
      <div className="relative rounded-lg px-4 pb-4 pt-16">
        {!!icon && <ResourceIcon icon={icon} />}
        <h3 className="mt-4 text-sm font-semibold leading-7 text-slate-900 dark:text-white">
          <Link href={resource.href}>
            <span className="absolute inset-0 rounded-lg" />
            {resource.name}
          </Link>
        </h3>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">{resource.description}</p>
      </div>
    </div>
  );
}

export function Resources() {
  return (
    <div className="my-16 xl:max-w-none">
      <Heading level={2} id="resources">
        Resources
      </Heading>
      <div className="not-prose mt-4 grid grid-cols-1 gap-8 border-t border-slate-900/5 pt-10 sm:grid-cols-2 xl:grid-cols-4 dark:border-white/5">
        {resources.map((resource) => (
          <Resource key={resource.href} resource={resource} />
        ))}
      </div>
    </div>
  );
}

export function ResourceGrid({ cols = 4, children }) {
  return (
    <div
      className={`not-prose mt-4 grid grid-cols-1 gap-8 border-t border-slate-900/5 pt-10 xl:max-w-none dark:border-white/5
      sm:grid-cols-${cols >= 2 ? 2 : cols} xl:grid-cols-${cols}`}
    >
      {children}
    </div>
  );
}
