import { extendTailwindMerge, twMerge } from 'tailwind-merge';

// Extend the default `tailwind-merge` configuration to include our custom `text-shadow` class group.
const customTwMerge = extendTailwindMerge({
  classGroups: {
    /**
     * Text Shadow
     * @see {@link file://../../../../apps/dashboard/src/app/globals.css}
     */
    'text-shadow': [{ 'text-shadow': ['', 'md', 'lg', 'none'] }],
  },
});

/**
 * Utility function for conditionally constructing `className` strings without style conflicts when
 * overriding Tailwind CSS classes.
 *
 * @see {@link https://github.com/dcastil/tailwind-merge/blob/main/docs/what-is-it-for.md} for
 * learning about style conflicts.
 *
 * @param inputs - A list of class names to merge.
 * @returns A string of merged class names.
 */
export function cn(...inputs: Parameters<typeof twMerge>): string {
  return customTwMerge(...inputs);
}
