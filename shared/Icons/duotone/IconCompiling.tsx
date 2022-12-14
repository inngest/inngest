import IconTheme from "./theme";

export function IconCompiling({
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
        <path
          d="M7.70710678,13.1213203 L9.12132033,11.7071068 C9.51184467,11.3165825 10.1450096,11.3165825 10.5355339,11.7071068 L11.9497475,13.1213203 C12.3402717,13.5118447 12.3402717,14.1450096 11.9497475,14.5355339 L10.5355339,15.9497475 C10.1450096,16.3402717 9.51184467,16.3402717 9.12132033,15.9497475 L7.70710678,14.5355339 C7.31658249,14.1450096 7.31658249,13.5118447 7.70710678,13.1213203 Z M7.70710678,5.12132035 L9.12132033,3.70710678 C9.51184467,3.31658249 10.1450096,3.31658249 10.5355339,3.70710678 L11.9497475,5.12132035 C12.3402717,5.51184463 12.3402717,6.14500961 11.9497475,6.53553391 L10.5355339,7.94974747 C10.1450096,8.34027176 9.51184467,8.34027176 9.12132033,7.94974747 L7.70710678,6.53553391 C7.31658249,6.14500961 7.31658249,5.51184463 7.70710678,5.12132035 Z"
          id="path-compiling-1"
        ></path>
        <filter
          x="-72.5%"
          y="-19.5%"
          width="245.0%"
          height="154.6%"
          filterUnits="objectBoundingBox"
          id="filter-compiling-2"
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
        id="Compiling"
        stroke="none"
        stroke-width="1"
        fill="none"
        fill-rule="evenodd"
      >
        <path
          d="M3.70710678,9.12132033 L5.12132035,7.70710678 C5.51184463,7.31658249 6.14500961,7.31658249 6.53553391,7.70710678 L7.94974747,9.12132033 C8.34027176,9.51184467 8.34027176,10.1450096 7.94974747,10.5355339 L6.53553391,11.9497475 C6.14500961,12.3402717 5.51184463,12.3402717 5.12132035,11.9497475 L3.70710678,10.5355339 C3.31658249,10.1450096 3.31658249,9.51184467 3.70710678,9.12132033 Z M11.7071068,9.12132033 L13.1213203,7.70710678 C13.5118447,7.31658249 14.1450096,7.31658249 14.5355339,7.70710678 L15.9497475,9.12132033 C16.3402717,9.51184467 16.3402717,10.1450096 15.9497475,10.5355339 L14.5355339,11.9497475 C14.1450096,12.3402717 13.5118447,12.3402717 13.1213203,11.9497475 L11.7071068,10.5355339 C11.3165825,10.1450096 11.3165825,9.51184467 11.7071068,9.12132033 Z"
          id="Combined-Shape"
          fill={theme.color}
          opacity={theme.opacity}
          fill-rule="nonzero"
        ></path>
        <g id="Combined-Shape" fill-rule="nonzero">
          <use
            fill="black"
            fill-opacity="1"
            filter="url(#filter-compiling-2)"
            xlinkHref="#path-compiling-1"
          ></use>
          <use fill="#FFFFFF" xlinkHref="#path-compiling-1"></use>
        </g>
      </g>
    </svg>
  );
}
