export function BoardBackground() {
  return (
    <div
      className="absolute inset-0 z-0 opacity-[0.03] pointer-events-none"
      style={{
        backgroundImage: `linear-gradient(#000 1px, transparent 1px), linear-gradient(90deg, #000 1px, transparent 1px)`,
        backgroundSize: "32px 32px",
      }}
    />
  );
}