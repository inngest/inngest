// Use the image version to bust social network's caches
const openGraphImageVersion = 2;

/*
 * Generates a URL to dynamically generate an open graph image for posts on social media
 * @see: /pages/api/og.tsx
 */
export const getOpenGraphImageURL = ({ title }: { title: string }) =>
  `https://www.inngest.com/api/og?title=${encodeURIComponent(
    title
  )}&v=${openGraphImageVersion}`;
