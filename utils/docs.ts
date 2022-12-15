/**
 * Docs Sections
 *
 * These are the sections that will be separated in the nav
 */
export enum Sections {
  default = "Default",
  reference = "Reference",
  cloud = "Inngest Cloud", // Hidden for now
}
const SectionOrder = [Sections.default, Sections.reference];
const HiddenSections = [Sections.cloud];

/**
 * Docs Table of Contents
 *
 * This is the basic order of categories for the docs navigation.
 */

const TOC = {
  "Getting started": 0,

  "Quick Start Tutorial": 5,
  "Writing functions": 10,
  "Sending events": 20,
  Deploying: 30,
  Frameworks: 40,
  Guides: 50,

  // "Events": 100,
  CLI: 200,

  // Reference
  SDK: 1000,
  Functions: 1010,
};

export type DocScope = {
  type: Sections;
  // If the slug contains a forward slahs (eg. foo/bar), this page will automatically
  // be nested under the page with a slug of "foo"
  slug: string;
  category: string;
  /** Featured image */
  image?: string;
  /** A meta description for the page itself */
  description?: string;
  /** Sub pages */
  pages?: DocScope[];
  // title is the title of the documentation page
  title: string;
  order: number;
  // if true, doc will be hidden on the nav, but will still be created as a page
  hide?: boolean;

  // reading is reading information automatically added when parsing content
  reading?: { text: string; time: number; words: number; minutes: number };
  // toc is the table of contents automatically added when parsing contnet
  toc?: Headings;
};

export type Heading = {
  order: number;
  title: string;
  slug: string;
  subheadings: [{ title: string; slug: string }];
};

export type Headings = {
  [title: string]: Heading;
};

export type Doc = {
  slug: string;
  content: string;
  scope: DocScope;
  type: Sections;
};

export type Category = {
  title: string;
  order: number;
  pages: DocScope[];
};

export type Categories = { [title: string]: Category };

export type Docs = {
  docs: { [slug: string]: Doc };
  // TODO: Is this an ordered set?
  slugs: string[];
  sections: { section: Sections; categories: Categories; hide: boolean }[];
};

export const getAllDocs = (() => {
  let memoizedDocs: Docs = {
    // docs maps docs by slug
    docs: {},
    slugs: [],
    sections: [],
  };

  return (): Docs => {
    if (
      memoizedDocs.slugs.length > 0 &&
      process.env.NODE_ENV === "production"
    ) {
      // memoizing in dev means you need to restart the server to see changes
      return memoizedDocs;
    }

    // Reset this locally during development.
    memoizedDocs = {
      // docs maps docs by slug
      docs: {},
      slugs: [],

      sections: [
        ...SectionOrder.map((section) => ({
          section,
          categories: {},
          hide: false,
        })),
        ...HiddenSections.map((section) => ({
          section,
          categories: {},
          hide: true,
        })),
      ],
    };

    const fs = require("fs");
    const matter = require("gray-matter");
    const readingTime = require("reading-time");

    // parseDir is given the current directory path, then returns a function which
    // can read all files from the given basepath and processes the input.
    const parseDir = (basepath: string, type: Sections) => (fname: string) => {
      const fullpath = basepath + fname;

      if (fs.statSync(fullpath).isDirectory()) {
        // recurse into this directory with a new parse function using the extended
        // path.
        fs.readdirSync(fullpath).forEach(parseDir(fullpath + "/", type));
        return;
      }

      const source = fs.readFileSync(fullpath);
      const { content, data: scope } = matter(source);

      const prefix = Object.keys(Sections).find((k) => Sections[k] === type);
      try {
        if (type !== Sections.default && scope.slug.indexOf(prefix) !== 0) {
          // Prefix for the section
          scope.slug = `${prefix}/${scope.slug}`;
        }
      } catch (err) {
        throw new Error(
          `Failed to read slug from file: ${fullpath}. Check the front-matter, it may be missing or incorrect.`
        );
      }

      memoizedDocs.slugs.push("/docs/" + scope.slug);
      memoizedDocs.docs[scope.slug] = {
        type,
        slug: scope.slug,
        content,
        scope: {
          type,
          ...scope,
          toc: getHeadings(content),
          reading: readingTime(content),
        },
      };
    };

    [...SectionOrder, ...HiddenSections].forEach((section) => {
      const key = Object.keys(Sections).find((k) => Sections[k] === section);
      const dir = section === Sections.default ? "docs" : key;
      const path = `./pages/docs/_${dir}/`;
      fs.readdirSync(path).forEach(parseDir(path, section));
    });

    [...SectionOrder, ...HiddenSections].forEach((section) => {
      const sectionObject = memoizedDocs.sections.find(
        (s) => s.section === section
      );
      const categories = sectionObject.categories;

      Object.values(memoizedDocs.docs).forEach((d: Doc) => {
        if (!d.scope.category) {
          // console.warn("no category for doc", JSON.stringify(d.scope));
          return;
        }
        // Only file for these sections
        if (d.type !== section) {
          return;
        }

        // Add category to list.
        if (!categories[d.scope.category]) {
          const order = TOC.hasOwnProperty(d.scope.category)
            ? TOC[d.scope.category]
            : 100;
          if (order === 100) {
            // console.warn("no order for category", d.scope.category);
          }
          categories[d.scope.category] = {
            title: d.scope.category,
            pages: [d.scope],
            order,
          };
        } else {
          categories[d.scope.category].pages.push(d.scope);
        }
      });
    });

    // And finally, order the slugs according to the category
    // order then the page order.
    //
    // In order to do this, we're going to sort categories and
    // then iterate through each doc within those categories
    // in order, pushing the slugs to a new sorted array.
    const sorted: Array<string> = [];

    memoizedDocs.sections.forEach((s) => {
      Object.values(s.categories)
        .sort((a, b) => a.order - b.order)
        .forEach((category) => {
          // Ensure the category index page is first in the overall order
          const categoryIndexPage = category.pages.find(
            (p) => p.category === p.title
          );
          if (categoryIndexPage) {
            sorted.push("/docs/" + categoryIndexPage.slug);
          }
          const nestedPages = category.pages.filter(
            (p) => p.category !== p.title
          );
          nestedPages.sort((a, b) => a.order - b.order);
          nestedPages.forEach((p) => sorted.push("/docs/" + p.slug));
        });
    });

    memoizedDocs.slugs = sorted;
    return memoizedDocs;
  };
})();

export const getDocs = (slug: string): Doc | undefined => {
  const docs = getAllDocs();
  return docs.docs[slug];
};

export const getHeadings = (content: string): Headings => {
  // Get headers for table of contents.
  const headings = {};
  let h2 = null; // store the current heading we're in
  let order = 0;

  (content.match(/^###? (.*)/gm) || []).forEach((heading) => {
    const title = heading.replace(/^###? /, "");
    if (heading.indexOf("## ") === 0) {
      h2 = title;
      headings[title] = { title, slug: toSlug(title), subheadings: [], order };
      order++;
      return;
    }
    // add this subheading to the current heading list.
    (headings[h2]?.subheadings || []).push({ title, slug: toSlug(title) });
  });
  return headings;
};

export const getHeadingsAsArray = (content: string): Heading[] => {
  const headingsObj = getHeadings(content);
  return Object.keys(headingsObj)
    .map((key) => headingsObj[key])
    .sort((a, b) => a.order - b.order);
};

const toSlug = (s: string) => {
  s = s.replace(/[^a-zA-Z0-9 :]/g, "");
  // rehype's `rehypeSlug` plugin converts "foo: one"  to "foo--one", and doesn't
  // remove multple slashes.  It does convert multiple spaces to just one slash.
  s = s.replace(/ +/g, "-");
  s = s.replace(/[:&]/g, "-");
  s = s.replace(/--/g, "-");
  return s.toLowerCase();
};
