<h1 align="center">Cue + TypeScript</h1>
<br />

**This package converts Cue types to TypeScript types, and soon vice-versa.**

Cue is a concise language for defining types and constraints in one file.  It's
best practice to have a single source of truth for your types.  This package
allows you to convert your Cue types to TypeScript for the frontend.

### Usage

This is a library for converting cue types, intended for use within a go
application.  Run the following command to install the package:

```
go get github.com/inngest/cuetypescript
```

## Examples


<table>
<tr><th>CUE</th><th>TypeScript</th></tr>
<tr>
<td>

```cue
#Post: {
	id:        string
	slug:      string
	title:     string
	subtitle?: string
	rating:    float & <=5
	category:  "tech" | "finance" | "hr"
	tags: [...string]
	references: [...{
		slug:  string
		title: string
	}]
}
```

</td>
<td>

```typescript
export const Category = {
  TECH: "tech",
  FINANCE: "finance",
  HR: "hr",
} as const;
export type Category = typeof Category[keyof typeof Category];

export interface Post {
  id: string;
  slug: string;
  title: string;
  subtitle?: string;
  rating: number;
  category: Category;
  tags: Array<string>;
  references: Array<{
    slug: string;
    title: string;
  }>;
};
```

</td>
</tr>
</table>

## Features

- Interface generation
- Type conversion and support
- Nested struct support
- Union support
- "Best practice" enum generation.  We create enums with both `const` and `type` values, allowing you to properly reference enum values via eg. `Category.TECH`.

In the future, we plan on adding:

- Function generation for checking and validating Cue constraints
- Default value generation and constructors
- Typescript to Cue support
