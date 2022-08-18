import { GetStaticProps } from "next";
import Button from "src/shared/Button";
import Nav from "src/shared/nav";
import { getExamples } from "./[example]";

interface Props {
  examples: Awaited<ReturnType<typeof getExamples>>;
}

export const getStaticProps: GetStaticProps = async () => {
  return { props: { examples: await getExamples() } };
};

export default function LibraryExamplesPage(props: Props) {
  return (
    <div>
      <Nav sticky nodemo />
      <div className="container mx-auto pt-32 pb-24 flex flex-row">
        <div className="text-center px-6 max-w-4xl mx-auto flex flex-col space-y-6">
          <h1>Examples</h1>
          <p className="subheading">...</p>
        </div>
      </div>

      <div className="container mx-auto max-w-lg mb-12 px-12">
        <input
          type="text"
          placeholder="Search..."
          className="w-full bg-gray-100 border border-gray-200"
        />
      </div>

      <div className="container mx-auto px-12 pb-12 grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-12">
        {props.examples.map((example) => (
          <div
            key={example.id}
            className="rounded-lg border border-gray-200 p-6 flex flex-col space-y-2"
          >
            <div className="font-semibold">{example.name}</div>
            <div>{example.description}</div>
            <Button
              kind="outline"
              size="medium"
              className="inline-block"
              href={`/examples/${example.id}?ref=examples`}
            >
              Explore
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
}
