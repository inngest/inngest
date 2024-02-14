import IconTheme from "./theme";

export function IconSendEvents({
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
          x="-33.7%"
          y="-26.5%"
          width="167.3%"
          height="174.3%"
          filterUnits="objectBoundingBox"
          id="send-events-a"
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
        <filter
          x="-83.3%"
          y="-29.7%"
          width="266.5%"
          height="183.3%"
          filterUnits="objectBoundingBox"
          id="send-events-c"
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
          d="m7.604 10.8107 8.7194-.8175-8.7193-.8174V5.6303a.3503.3503 0 0 1 .494-.3194l9.6953 4.363a.3503.3503 0 0 1 0 .6387l-9.6953 4.363a.3503.3503 0 0 1-.494-.3195v-3.5454Z"
          id="send-events-b"
        />
        <path
          d="M4.802 12.7953h.7006a.7005.7005 0 0 1 0 1.401H4.802a.7005.7005 0 1 1 0-1.401ZM2.7005 9.2927h2.802a.7005.7005 0 0 1 0 1.401h-2.802a.7005.7005 0 0 1 0-1.401ZM4.802 5.7902h.7006a.7005.7005 0 1 1 0 1.401H4.802a.7005.7005 0 1 1 0-1.401Z"
          id="send-events-d"
        />
      </defs>
      <g fill="none" fillRule="nonzero">
        <use
          fill="#000"
          filter="url(#send-events-a)"
          xlinkHref="#send-events-b"
        />
        <use
          fill={theme.color}
          opacity={theme.opacity}
          xlinkHref="#send-events-b"
        />
        <use
          fill="#000"
          filter="url(#send-events-c)"
          xlinkHref="#send-events-d"
        />
        <use fill="#FFF" xlinkHref="#send-events-d" />
      </g>
    </svg>
  );
}
