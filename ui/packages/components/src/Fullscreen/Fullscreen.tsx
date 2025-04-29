import { Portal } from '@radix-ui/react-portal';

/**
 * A full screen portal component that side steps all intervening z-indexes
 */
export const Fullscreen = ({
  fullScreen = false,
  children,
}: {
  fullScreen?: boolean;
  children: React.ReactNode;
}) => {
  return fullScreen ? <Portal className="absolute inset-0 z-[100]">{children}</Portal> : children;
};
