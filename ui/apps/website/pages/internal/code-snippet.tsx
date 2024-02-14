import { useState } from 'react';
import styled from '@emotion/styled';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { atomOneDark as syntaxThemeDark } from 'react-syntax-highlighter/dist/cjs/styles/hljs';
import { removeLeadingSpaces } from 'src/shared/CodeWindow';

const defaultCode = `
  import { inngest } from "./client";

  export default inngest.createFunction(
    { id: "signup" },
    { event: "app/user.signup" },
    async ({ event }) => {
      // ...
    }
  );
`;

export async function getStaticProps() {
  return {
    props: {
      designVersion: '2',
    },
  };
}

export default function CodeSnippet() {
  const [code, setCode] = useState<string>(defaultCode);
  const [filename, setFilename] = useState<string>('function.ts');
  // const [theme, setTheme] = useState<"dark" | "light">("dark");
  const [backgroundColor, setBackgroundColor] = useState<string>('#080D19');
  const [borderColor, setBorderColor] = useState<string>('#94a3b8');
  const [zoom, setZoom] = useState<string>('150');
  return (
    <Container style={{ backgroundColor }}>
      <div className="py-36">
        {/* <CodeWindow
          snippet={code}
          filename={filename}
          className="inline-block my-8 pr-8 border-2"
          theme={theme}
          style={{
            borderColor,
            transform: `scale(${zoom}%)`,
          }}
        /> */}
        <div
          className="inline-block max-w-full overflow-hidden overflow-x-scroll rounded-lg border border-slate-800/60 bg-slate-950/80 shadow-lg backdrop-blur-md"
          style={{ transform: `scale(${zoom}%)` }}
        >
          <h6 className="w-full border-b border-slate-800/50 bg-slate-950/50 py-1.5 text-center text-xs text-slate-300">
            {filename}
          </h6>

          <SyntaxHighlighter
            language="typescript"
            showLineNumbers={false}
            style={syntaxThemeDark}
            codeTagProps={{ className: 'code-window' }}
            customStyle={{
              backgroundColor: 'transparent',
              fontSize: '0.7rem',
              padding: '1.5rem',
            }}
          >
            {removeLeadingSpaces(code)}
          </SyntaxHighlighter>
        </div>
      </div>
      <div className="mt-16 w-full text-xs" style={{ backgroundColor: '#080D19', color: '#fff' }}>
        <h2>Controls</h2>
        <div className="my-2">
          <label>
            Color{' '}
            <input
              type="color"
              name="color"
              defaultValue={backgroundColor}
              onChange={(e) => {
                setBackgroundColor(e.target.value);
              }}
              className="bg-slate-800 p-0"
            />
          </label>
        </div>
        <div className="my-2">
          <label>
            Border Color{' '}
            <input
              type="color"
              name="borderColor"
              defaultValue={borderColor}
              onChange={(e) => {
                setBorderColor(e.target.value);
              }}
              className="bg-slate-800 p-0"
            />
          </label>
        </div>
        <div className="my-2">
          <label>
            Zoom{' '}
            <input
              type="number"
              className="bg-slate-800 text-xs"
              style={{ border: '1px solid #94a3b8' }}
              onChange={(e) => setZoom(e.target.value)}
              defaultValue={zoom}
            />
          </label>
        </div>
        <div className="my-2">
          <label>
            Filename{' '}
            <input
              type="text"
              className="bg-slate-800 text-xs"
              style={{ border: '1px solid #94a3b8' }}
              onChange={(e) => setFilename(e.target.value)}
              defaultValue={filename}
            />
          </label>
        </div>
        <div className="my-2">
          <label>
            Code{' '}
            <textarea
              className="h-64 w-full bg-slate-800 font-mono text-xs"
              style={{ border: '1px solid #94a3b8', fontFamily: 'monospace' }}
              onChange={(e) => setCode(e.target.value)}
              defaultValue={removeLeadingSpaces(code)}
            ></textarea>
          </label>
        </div>
      </div>
    </Container>
  );
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  width: 100%;
  padding: 2rem 1rem;
  background: #fff;
`;
