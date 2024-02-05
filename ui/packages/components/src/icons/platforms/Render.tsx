export function IconRender({ className, size = 18 }: { className?: string; size?: number }) {
  return (
    <svg
      className={className}
      fill="currentColor"
      height={size}
      viewBox="0 0 171 171"
      width={size}
      xmlns="http://www.w3.org/2000/svg"
    >
      <path d="M111.3 0v22.6h22.6V0h-20.6zM53.5 0A53 53 0 0 0 4.2 32.7 53 53 0 0 0 0 53.5v80.4h22.6V53.2a32 32 0 0 1 31-30.6h43.2V0h-2zm94.9 74.2v22.6H171V74.2h-20.6zm0-37.1v22.6H171V37.1h-20.6zm2-37.1h-2v22.6H171V0h-2zm-2 111.3v22.6H171v-22.6h-20.6zm0 37.1V171H171v-22.6h-20.6zm-37.1 0V171h22.6v-22.6h-20.6zm-37.1 0V171h22.6v-22.6H76.2zm-37.1 0V171h22.6v-22.6H39.1zm-37.1 0V171h22.6v-22.6H2z" />
    </svg>
  );
}
