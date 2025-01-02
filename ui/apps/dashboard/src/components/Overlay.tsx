type OverlayProps = {
  children?: React.ReactNode;
};

export default function Overlay({ children }: OverlayProps) {
  return (
    <div className="bg-canvasBase/30 absolute z-[2] h-full w-full backdrop-blur-[2px]">
      {children}
    </div>
  );
}
