export {};

declare global {
  interface Window {
    Inngest: any;
    _inngestQueue: { [key: string]: any }[];
  }
}

declare module "deterministic-split";
