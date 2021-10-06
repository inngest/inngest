import "src/wasm/go-wasm-exec";

type ParseCueFn = (input: string) => string | object;
type SerializeCueFn = (input: string) => string;

declare global {
  interface Window {
    parseCue?: ParseCueFn;
    serializeCue?: SerializeCueFn;
    Go: any;
  }
}

let loaded = false;
export async function init() {
  // only run once.
  if (loaded) return;

  const go = new window.Go();
  let result = await WebAssembly.instantiateStreaming(
    fetch("/wasm/cue.wasm"),
    go.importObject
  );
  go.run(result.instance);
}
