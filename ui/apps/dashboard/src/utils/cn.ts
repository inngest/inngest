import { clsx, type ClassValue } from 'clsx';
import { extendTailwindMerge } from 'tailwind-merge';

// Extend the default `tailwind-merge` configuration to include our custom `text-shadow` class group.
const customTwMerge = extendTailwindMerge({
  classGroups: {
    /**
     * Text Shadow
     * @see {@link file://../app/globals.css}
     */
    'text-shadow': [{ 'text-shadow': ['', 'md', 'lg', 'none'] }],
  },
});

/**
 * Utility function for conditionally constructing `className` strings without style conflicts when
 * overriding Tailwind CSS classes.
 *
 * @see {@link https://github.com/lukeed/clsx#usage} for examples of all supported input types.
 * @see {@link https://github.com/dcastil/tailwind-merge/blob/main/docs/what-is-it-for.md} for
 * learning about style conflicts.
 *
 * @param inputs - A list of class names to merge.
 * @returns A string of merged class names.
 */
export default function cn(...inputs: ClassValue[]): string {
  return customTwMerge(clsx(...inputs));
}
