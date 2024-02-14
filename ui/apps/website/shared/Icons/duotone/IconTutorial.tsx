import IconTheme from "./theme";

export function IconTutorial({
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
      version="1.1"
    >
      <defs>
        <path
          d="M16.6146966,6.58452753 L11.2212056,3.6419415 C10.4604285,3.2260195 9.54024454,3.2260195 8.77946751,3.6419415 L3.38597651,6.58452753 C2.83979225,6.88265275 2.5,7.45523716 2.5,8.07748784 C2.5,8.69973851 2.83979225,9.27232292 3.38597651,9.57044815 L8.77946751,12.5122008 C9.54024454,12.9281228 10.4604285,12.9281228 11.2212056,12.5122008 L16.0421798,9.88212392 L16.0421798,13.3305581 C16.0421798,13.6757461 16.32201,13.9555763 16.6671981,13.9555763 C17.0123862,13.9555763 17.2922164,13.6757461 17.2922164,13.3305581 L17.2922164,8.88959491 C17.7414498,8.06595461 17.4381576,7.03408951 16.6146966,6.58452753 L16.6146966,6.58452753 Z"
          id="path-1"
        ></path>
        <filter
          x="-23.3%"
          y="-23.5%"
          width="146.7%"
          height="165.9%"
          filterUnits="objectBoundingBox"
          id="filter-2"
        >
          <feOffset
            dx="0"
            dy="1"
            in="SourceAlpha"
            result="shadowOffsetOuter1"
          ></feOffset>
          <feGaussianBlur
            stdDeviation="1"
            in="shadowOffsetOuter1"
            result="shadowBlurOuter1"
          ></feGaussianBlur>
          <feColorMatrix
            values="0 0 0 0 0   0 0 0 0 0   0 0 0 0 0  0 0 0 0.268793706 0"
            type="matrix"
            in="shadowBlurOuter1"
          ></feColorMatrix>
        </filter>
      </defs>
      <g
        id="Tutorial"
        stroke="none"
        strokeWidth="1"
        fill="none"
        fillRule="evenodd"
      >
        <path
          d="M8.77946751,12.5122008 L5.01102401,10.4563074 L5.00019036,10.4563074 L5.00019036,13.9139085 C5.00375692,14.4963024 5.30577944,15.0361676 5.80021375,15.3439503 C8.32347235,17.1047771 11.6772007,17.1047771 14.2004593,15.3439503 C14.6948936,15.0361676 14.9969161,14.4963024 15.0004827,13.9139085 L15.0004827,10.4563074 L14.9896491,10.4563074 L11.2212056,12.5122008 C10.4604285,12.9281228 9.54024454,12.9281228 8.77946751,12.5122008 Z"
          id="Path"
          fill={theme.color}
          opacity={theme.opacity}
          fillRule="nonzero"
        ></path>
        <g id="Path" fillRule="nonzero">
          <use
            fill="black"
            fillOpacity="1"
            filter="url(#filter-2)"
            xlinkHref="#path-1"
          ></use>
          <use fill="#FFFFFF" xlinkHref="#path-1"></use>
        </g>
      </g>
    </svg>
  );
}
