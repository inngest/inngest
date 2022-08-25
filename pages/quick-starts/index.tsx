import Fuse from "fuse.js";
import { GetStaticProps } from "next";
import { queryTypes, useQueryState } from "next-usequerystate";
import Link from "next/link";
import { useMemo } from "react";
import Footer from "src/shared/footer";
import Nav from "src/shared/nav";
import { reqWithSchema } from "src/utils/fetch";
import { z } from "zod";

const miniCache: Record<string, any> = {};

interface Props {
  examples: Awaited<ReturnType<typeof getExamples>>;
}

export default function LibraryExamplesPage(props: Props) {
  const [search, setSearch] = useQueryState(
    "search",
    queryTypes.string.withDefault("")
  );
  const trimmedSearch = useMemo(() => search.trim(), [search]);
  const [rawActiveTags, setActiveTags] = useQueryState(
    "tags",
    queryTypes.array(queryTypes.string).withDefault([])
  );

  const activeTags = useMemo(() => {
    return rawActiveTags.filter((tag) => Boolean(tag.trim()));
  }, [rawActiveTags]);

  const filteredExamples = useMemo(() => {
    if (activeTags.length) {
      return props.examples.filter((example) =>
        activeTags.every((tag) => example.tags.includes(tag))
      );
    }

    return props.examples;
  }, [activeTags, props.examples]);

  const fuse = useMemo(() => {
    return new Fuse(filteredExamples, {
      keys: ["name", "description", "tags"],
      findAllMatches: true,
    });
  }, [filteredExamples]);

  const examples = useMemo(() => {
    if (!trimmedSearch) {
      return filteredExamples;
    }

    return fuse.search(trimmedSearch).map(({ item }) => item);
  }, [fuse, trimmedSearch, filteredExamples]);

  return (
    <div>
      <Nav sticky nodemo />
      <div className="hero-gradient">
        <div className="container mx-auto pt-32 pb-24 flex flex-row">
          <div className="text-center px-6 max-w-4xl mx-auto flex flex-col space-y-6">
            <h1>
              Start building <span className="gradient-text">in seconds</span>
            </h1>
            <p className="subheading max-w-lg mx-auto">
              Use our ready-to-deploy pre-built functions for inspiration or as
              a starting point for your own project.
            </p>
          </div>
        </div>
        <div className="max-w-lg px-12 pb-6 mx-auto">
          <input
            type="text"
            placeholder="Search..."
            className="w-full bg-white shadow-lg border border-gray-200"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      </div>

      <div className="background-grid-texture">
        <div className="px-12 py-6 max-auto flex flex-row items-center justify-center">
          <div className="italic text-xs text-gray-500 flex flex-row items-center justify-center">
            Showing {!trimmedSearch && !activeTags.length ? "all " : ""}
            {examples.length} quick-starts
            {trimmedSearch ? ` matching "${trimmedSearch}"` : ""}
            {activeTags.length ? (
              <>
                {" "}
                with tags:{" "}
                <span className="flex flex-row gap-1">
                  {activeTags.map((tag) => (
                    <Tag
                      key={tag}
                      name={tag}
                      onSelect={() => {
                        setActiveTags((currTags) => {
                          if (currTags.includes(tag)) {
                            return currTags.filter((t) => t !== tag);
                          }

                          return [...currTags, tag];
                        });
                      }}
                    />
                  ))}
                </span>
              </>
            ) : (
              ""
            )}
          </div>
        </div>
        <div className="container mx-auto px-12 pb-12 pt-6 grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-12">
          {examples.map((example) => (
            <div className="flex flex-col w-full h-full">
              <Link
                key={example.id}
                href={`/quick-starts/${example.id}?ref=quick-starts`}
                passHref
              >
                <a className="rounded-lg border border-gray-200 p-6 flex flex-col space-y-2 bg-white transition-all transform hover:scale-105 hover:shadow-lg flex-1">
                  <div className="text-black">{example.name}</div>
                  {example.description ? (
                    <div className="text-sm text-gray-500">
                      {example.description}
                    </div>
                  ) : null}
                  {example.tags.length ? (
                    <div className="flex flex-row flex-wrap gap-1">
                      {example.tags.map((tag) => (
                        <Tag
                          key={tag}
                          name={tag}
                          onSelect={() => {
                            setActiveTags((currTags) => {
                              if (currTags.includes(tag)) {
                                return currTags.filter((t) => t !== tag);
                              }

                              return [...currTags, tag];
                            });
                          }}
                        />
                      ))}
                    </div>
                  ) : null}
                  <div className="flex-1" />
                  <a className="text-blue-500 font-semibold text-right">
                    Explore →
                  </a>
                </a>
              </Link>
            </div>
          ))}
        </div>
      </div>
      <Footer />
    </div>
  );
}

interface TagProps {
  name: string;
  onSelect?: () => void;
}

const Tag: React.FC<TagProps> = ({ name, onSelect }) => {
  return (
    <div
      key={name}
      className="uppercase text-[0.7rem] px-1 bg-blue-100/50 text-blue-500 hover:bg-blue-100 hover:text-blue-600 transition rounded cursor-pointer"
      onClick={(e) => {
        e.stopPropagation();
        e.preventDefault();
        onSelect();
      }}
    >
      {name}
    </div>
  );
};

export const getStaticProps: GetStaticProps = async () => {
  return {
    props: {
      examples: await getExamples(),
      meta: {
        title: "Quickstarts - Start building in seconds",
        description:
          "Use our ready-to-deploy, pre-built functions for inspiration or as a starting point for your own project.",
      },
    },
  };
};

const CommitsResponseObj = z.object({
  commit: z.object({ tree: z.object({ url: z.string() }) }),
});
type CommitsResponse = z.infer<typeof CommitsResponseObj>;

export const getExamples = async () => {
  let latestCommit: CommitsResponse;
  try {
    latestCommit = await reqWithSchema(
      "https://api.github.com/repos/inngest/inngest/commits/main",
      CommitsResponseObj,
      miniCache
    );
  } catch (err) {
    // Use https://github.com/settings/tokens to generate token w/ "public_repo" and "repo:status" scopes
    // and set locally in .env.local file
    throw new Error(
      "Invalid response from Github API. Potential rate limit issue, set GITHUB_TOKEN with personal access token to proceed"
    );
  }

  const commitContents = await reqWithSchema(
    latestCommit.commit.tree.url,
    z.object({
      tree: z.array(treeSchema),
    }),
    miniCache
  );

  const examplesPath = commitContents.tree.find(
    ({ path, type }) => path === "examples" && type === "tree"
  );

  if (!examplesPath?.url) {
    throw new Error("Could not find examples path");
  }

  const examplesTree = await reqWithSchema(
    examplesPath.url,
    z.object({
      tree: z.array(treeSchema),
    }),
    miniCache
  );

  const exampleNodes = examplesTree.tree.filter(({ type }) => type === "tree");

  const examples = await Promise.all(
    exampleNodes.map(async (node) => {
      const example = await reqWithSchema(
        node.url,
        z.object({ tree: z.array(treeSchema) }),
        miniCache
      );

      const inngestJsonNode = example.tree.find(
        ({ path, type }) => path === "inngest.json" && type === "blob"
      );

      if (!inngestJsonNode?.url) {
        throw new Error("Could not find inngest.json in example");
      }

      const inngestJsonRaw = await reqWithSchema(
        inngestJsonNode.url,
        blobSchema,
        miniCache
      );

      return z
        .object({
          id: z.string(),
          tree: z.array(treeSchema),
          name: z.string(),
          description: z.string().optional(),
          tags: z.array(z.string()).optional(),
        })
        .parse({
          ...JSON.parse(
            inngestJsonRaw.encoding === "base64"
              ? Buffer.from(inngestJsonRaw.content, "base64").toString()
              : inngestJsonRaw.content
          ),
          id: node.path,
          tree: example.tree,
        });
    })
  );

  return examples;
};

const treeSchema = z.object({
  path: z.string(),
  sha: z.string().or(z.null()),
  type: z.union([z.literal("blob"), z.literal("tree"), z.literal("commit")]),
  url: z.string(),
});

export const blobSchema = z.object({
  content: z.string(),
  encoding: z.union([z.literal("utf-8"), z.literal("base64")]),
});
