export type PageProps = {
  htmlClassName?: string;
  designVersion?: "2";
  meta?: {
    title?: string;
    description?: string;
    image?: string;
    // Disable auto-injecting meta tags
    disabled?: boolean;
  };
};
