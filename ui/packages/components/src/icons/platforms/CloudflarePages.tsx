export function IconCloudflarePages({
  className,
  size = 18,
}: {
  className?: string;
  size?: number;
}) {
  return (
    <svg
      className={className}
      height={size}
      viewBox="0 0 64 64"
      width={size}
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="none" fill-rule="nonzero">
        <path
          fill="currentColor"
          d="M41.94 8H56l2 2v44l-2 2H36.84l.97-1.5h17.57l1.12-1.12V10.62L55.38 9.5H43.26l-1.43 7.39H40.3l1.37-7.46.28-1.43zM8 56l-2-2V10l2-2h19.9L26.9 9.5H8.62L7.5 10.62v42.76l1.12 1.12H23.1l-.24 1.5H8zm3-5h8.5l-.3 1.5H10l-.5-.5v-9l1.5 3v5zm34 0l1.5 1.5H39l1-1.5h5z"
        />
        <path
          fill="currentColor"
          d="M28.67 38H15l-1.66-3.12 23-34 3.62 1.5L35.42 26H49l1.68 3.09-22 34-3.66-1.4L28.67 38zM11.5 15a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm4 0a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm4 0a1.5 1.5 0 110-3 1.5 1.5 0 010 3z"
        />
      </g>
    </svg>
  );
}
