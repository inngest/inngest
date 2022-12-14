import IconTheme from "./theme";
export function IconBackgroundTasks({
  size = 20,
  className = "",
  color = "transparent",
}) {
  const theme = IconTheme(color);
  return (
    <svg
      width={size}
      height={size}
      className={className}
      viewBox="0 0 20 20"
      xmlns="http://www.w3.org/2000/svg"
      xmlnsXlink="http://www.w3.org/1999/xlink"
    >
      <defs>
        <filter
          x="-64.8%"
          y="-46.3%"
          width="229.5%"
          height="229.5%"
          filterUnits="objectBoundingBox"
          id="background-tasks-a"
        >
          <feOffset dy="1" in="SourceAlpha" result="shadowOffsetOuter1" />
          <feGaussianBlur
            stdDeviation="1"
            in="shadowOffsetOuter1"
            result="shadowBlurOuter1"
          />
          <feColorMatrix
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.268793706 0"
            in="shadowBlurOuter1"
          />
        </filter>
        <path
          d="M7.3077 11.5385v-3.077a1.02 1.02 0 0 1 1.1538-1.1538h3.077a1.02 1.02 0 0 1 1.1538 1.1538v3.077a1.02 1.02 0 0 1-1.1538 1.1538h-3.077a1.02 1.02 0 0 1-1.1538-1.1538Z"
          id="background-tasks-b"
        />
      </defs>
      <g fill="none" fill-rule="nonzero">
        <path
          d="M4.6154 13.077V6.923A2.04 2.04 0 0 1 6.923 4.6155h6.1538a2.04 2.04 0 0 1 2.3077 2.3077v6.1538a2.04 2.04 0 0 1-2.3077 2.3077H6.9231a2.04 2.04 0 0 1-2.3077-2.3077Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
        <use fill="#000" filter="url(#background-tasks-a)" xlinkHref="#b" />
        <use fill="#FFF" xlinkHref="#background-tasks-b" />
        <path
          d="M17.5 7.6923a.5762.5762 0 0 1-.577.577h-1.5384v-1.154h1.5385a.5762.5762 0 0 1 .5769.577Zm-.577 4.0385h-1.5384v1.1538h1.5385a.577.577 0 1 0 0-1.1538ZM3.077 7.1154a.577.577 0 1 0 0 1.1538h1.5384V7.1154H3.0769Zm0 4.6154a.577.577 0 0 0 0 1.1538h1.5384v-1.1538H3.0769ZM12.3076 2.5a.5808.5808 0 0 0-.577.577v1.5384h1.154V3.0769a.5762.5762 0 0 0-.577-.5769Zm-4.6154 0a.5808.5808 0 0 0-.577.577v1.5384h1.154V3.0769a.5762.5762 0 0 0-.577-.5769Zm4.0385 12.8846v1.5385a.577.577 0 1 0 1.1538 0v-1.5385h-1.1538Zm-4.6154 0v1.5385a.577.577 0 0 0 1.1538 0v-1.5385H7.1154Z"
          fill="#FFF"
        />
      </g>
    </svg>
  );
}
