export const uuid = (userId?: string) => {
  let entropy = 0;
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (entropy + Math.random() * 16) % 16 | 0;
    entropy = Math.floor(entropy / 16);
    // We need to replace 'x' wholly and 'y' with the UUID variant
    return (c === "x" ? r : (r & 0x3) | 0x8).toString(16);
  });
};

export default uuid;
