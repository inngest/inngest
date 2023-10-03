export function IconStatusCircleMoon({ className, title }: { className?: string; title?: string }) {
  return (
    <svg
      width="28"
      height="28"
      viewBox="0 0 30 30"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      color="#38BDF8"
    >
      <title>{title}</title>
      <g>
        <path
          d="M15 26C21.0751 26 26 21.0751 26 15C26 8.92487 21.0751 4 15 4C8.92487 4 4 8.92487 4 15C4 21.0751 8.92487 26 15 26Z"
          fill="currentColor"
        />

        <svg
          x="5"
          y="5"
          width="20"
          height="20"
          viewBox="0 0 14 14"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          stroke="#1b5d7a"
          stroke-width="1"
        >
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M5.21902 1.40287C5.38002 1.53146 5.45091 1.74234 5.40029 1.94207C5.30249 2.32794 5.25039 2.73255 5.25039 3.15008C5.25039 5.85628 7.4442 8.05008 10.1504 8.05008C10.9509 8.05008 11.7051 7.85854 12.3711 7.51927C12.5547 7.42574 12.776 7.44826 12.937 7.57685C13.098 7.70544 13.1689 7.91632 13.1183 8.11605C12.4652 10.693 10.1312 12.6001 7.35039 12.6001C4.0643 12.6001 1.40039 9.93617 1.40039 6.65008C1.40039 4.33408 2.72373 2.32808 4.65309 1.34529C4.83669 1.25176 5.05802 1.27428 5.21902 1.40287Z"
            fill="white"
          />
        </svg>
      </g>
    </svg>
  );
}
