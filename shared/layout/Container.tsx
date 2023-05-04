export default function Container({ children, className = "", style = {} }) {
  return (
    <div
      className={`max-w-container-desktop m-auto px-6 md:px-10 lg:px-20 relative z-10 ${className}`}
      style={style}
    >
      {children}
    </div>
  );
}
