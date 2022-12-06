export default function IconDocs() {
  return (
    <svg width="42" height="42" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <filter
          x="-50%"
          y="-50%"
          width="200%"
          height="200%"
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
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.210227273 0"
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
        transform="translate(9 9)"
        fill="none"
        fillRule="evenodd"
      >
        <path d="M0 0h24v24H0z" />
        <path
          d="M13.6855 18.7082C15.9114 17.819 18.6829 17.2496 22 17V5.5063a.5.5 0 0 0-.5-.5c-.0042 0-.0084 0-.0127.0002C18.6583 5.078 15.8291 5.7426 13 7v11.2439a.5.5 0 0 0 .6855.4643Z"
          fill="#FFF"
          fillRule="nonzero"
        />
        <path
          d="M10.3145 18.7082C8.0886 17.819 5.317 17.2496 2 17V5.5063a.5.5 0 0 1 .5126-.4998C5.3418 5.078 8.171 5.7426 11 7v11.2439a.5.5 0 0 1-.6855.4643Z"
          fill="#FFF"
          fillRule="nonzero"
          opacity=".3"
        />
      </g>
    </svg>
  );
}
