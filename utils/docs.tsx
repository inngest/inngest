export type DocScope = {
  category?: string;
  title?: string;
  position?: number;
  reading: { text: string, time: number, words: number, minutes: number };

  toc?: Headings;
}

type Headings = {
  [title: string]: {
    order: number;
    title: string;
    slug: string;
    subheadings: [{ title: string; slug: string; }]
  }
};

type Doc = {
  slug: string,
  content: string,
  scope: DocScope,
}

type Docs = {
  docs: { [slug: string]: Doc }
  slugs: string[];
}

export const getAllDocs = (() => {
  let memoizedDocs = {
    // docs maps docs by slug 
    docs: {},
    slugs: []
  }

  return (): Docs => {
    if (memoizedDocs.slugs.length > 0) {
      return memoizedDocs;
    }
    
    const fs = require('fs');
    const matter = require('gray-matter');
    const readingTime = require('reading-time');

    fs.readdirSync("./pages/docs/_docs/").filter((fname: string) => {
      const source = fs.readFileSync("./pages/docs/_docs/" + fname);
      const { content, data: scope } = matter(source)

      const slug = `/docs/${fname.replace(/.mdx?/, "")}`
      memoizedDocs.slugs.push(slug);
      memoizedDocs.docs[slug] = {
        slug,
        content,
        scope: {
          ...scope,
          toc: getHeadings(content),
          reading: readingTime(content),
        },
      }
    });

    // TODO: Create categories, etc.


    return memoizedDocs;
  }
})()

export const getDocs = (slug: string): Doc | undefined => {
  const docs = getAllDocs();
  return docs.docs[slug];
}

const getHeadings = (content: string) => {
  // Get headers for table of contents.
  const slugify = require('slugify');
  const headings = {};
  let h2 = null; // store the current heading we're in
  let order = 0;

  (content.match(/^###? (.*)/gm) || []).forEach(heading => {
    const title = heading.replace(/^###? /, "");
    if (heading.indexOf("## ") === 0) {
      h2 = title;
      headings[title] = { title, slug: slugify(title).toLowerCase(), subheadings: [], order }
      order++
      return;
    }
    // add this subheading to the current heading list.
    headings[h2].subheadings.push({ title, slug: slugify(title) });
  });
  return headings;
}
