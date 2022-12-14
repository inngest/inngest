import IconTheme from "./theme";

export function IconScheduled({
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
          x="-96%"
          y="-47.6%"
          width="292%"
          height="233.2%"
          filterUnits="objectBoundingBox"
          id="scheduled-a"
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
          d="M9.9753 6.7h.0564a.3333.3333 0 0 1 .3316.3002l.3034 3.0331 2.1653 1.2374A.3333.3333 0 0 1 13 11.56V11.7a.2546.2546 0 0 1-.3216.2456l-3.0793-.8398a.3333.3333 0 0 1-.2446-.3471l.2885-3.751A.3333.3333 0 0 1 9.9753 6.7Z"
          id="scheduled-b"
        />
      </defs>
      <g fill="none" fill-rule="nonzero">
        <path
          d="M6.7621 14.605c.352.2693.7386.4958 1.1519.6716l-.6526 1.1304a.6667.6667 0 1 1-1.1547-.6667l.6554-1.1353Zm5.329.6694a5.3287 5.3287 0 0 0 1.1512-.6728l.6457 1.1185a.6667.6667 0 1 1-1.1547.6667l-.6422-1.1124Z"
          fill="#FFF"
        />
        <path
          d="M10 15.7c-2.9455 0-5.3333-2.3878-5.3333-5.3333 0-2.9456 2.3878-5.3334 5.3333-5.3334 2.9455 0 5.3333 2.3878 5.3333 5.3334 0 2.9455-2.3878 5.3333-5.3333 5.3333Zm4.7125-11.8306 1.1663 1.1662c.3905.3905.3905 1.0237 0 1.4142-.3906.3905-1.0237.3905-1.4143 0l-1.1662-1.1662c-.3905-.3905-.3905-1.0237 0-1.4142.3906-.3905 1.0237-.3905 1.4142 0Zm-9.18-.248c.3904-.3905 1.0236-.3905 1.4141 0 .3906.3905.3906 1.0237 0 1.4142L5.5324 6.4498c-.3905.3905-1.0237.3905-1.4142 0-.3905-.3905-.3905-1.0237 0-1.4142l1.4142-1.4142Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
        <use fill="#000" filter="url(#scheduled-a)" xlinkHref="#b" />
        <use fill="#FFF" xlinkHref="#scheduled-b" />
      </g>
    </svg>
  );
}
