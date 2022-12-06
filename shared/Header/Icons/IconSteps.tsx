export default function IconSteps() {
  return (
    <svg width="38" height="36">
      <defs>
        <filter
          x="-61.5%"
          y="-70.6%"
          width="223.1%"
          height="241.2%"
          filterUnits="objectBoundingBox"
          id="a"
        >
          <feOffset dy="2" in="SourceAlpha" result="shadowOffsetOuter1" />
          <feGaussianBlur
            stdDeviation="2"
            in="shadowOffsetOuter1"
            result="shadowBlurOuter1"
          />
          <feColorMatrix
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.219187063 0"
            in="shadowBlurOuter1"
            result="shadowMatrixOuter1"
          />
          <feMerge>
            <feMergeNode in="shadowMatrixOuter1" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>
      <g
        filter="url(#a)"
        transform="translate(9 9.7782)"
        fill="#FFF"
        fillRule="nonzero"
      >
        <circle cx="3.75" cy="3.5" r="3.5" />
        <circle cx="15.75" cy="13.5" r="3.5" />
        <path
          d="M18.75 4.25h-6a.75.75 0 1 1 0-1.5h6a.75.75 0 1 1 0 1.5ZM6.75 14.25h-6a.75.75 0 1 1 0-1.5h6a.75.75 0 1 1 0 1.5Z"
          opacity=".6"
        />
      </g>
    </svg>
  );
}
