type OverlayProps = {
  children?: React.ReactNode;
};

export default function Overlay({ children }: OverlayProps) {
  return (
    <div className="absolute z-[2] h-full w-full bg-white/30 backdrop-blur-[2px]">{children}</div>
  );
}
