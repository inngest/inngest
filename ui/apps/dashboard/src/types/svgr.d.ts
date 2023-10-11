declare type SVGComponent = React.FC<React.SVGProps<SVGSVGElement>>;

declare module '*.svg' {
  const ReactComponent: SVGComponent;
  export default ReactComponent;
}

declare module '*.svg?url' {
  const content: string;
  export default content;
}
