export default function PageContainer({ children }) {
  return (
    <div className="relative bg-slate-1000 font-sans">
      <div
        style={{
          background: "radial-gradient(circle at center, #13123B, #08090d)",
        }}
        className="absolute w-[200vw] -translate-x-1/2 -translate-y-1/2 h-[200vw] rounded-full blur-lg opacity-90"
      ></div>

      {children}
    </div>
  );
}
