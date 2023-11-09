import Link from "next/link";
import clsx from "clsx";

import Container from "../layout/Container";
import Heading from "./Heading";

const numbers = [
  {
    title: "250m+",
    subtitle: "Executions every month",
  },
  {
    title: "50k+",
    subtitle: "Deployed apps every month",
  },
  {
    title: "99.99%",
    subtitle: "Our Event API's uptime",
  },
];

export default function EnterpriseTrust() {
  return (
    <Container className="mt-12">
      <Heading
        title="Scale with confidence"
        lede={<></>}
        className="text-center"
      />

      <div className="my-16 mx-auto max-w-6xl grid md:grid-cols-3">
        {numbers.map(({ title, subtitle }) => (
          <h3 className="flex flex-col items-center gap-2 " key={title}>
            <span
              className="text-7xl font-extrabold tracking-tight bg-gradient-to-br from-white to-slate-300 bg-clip-text text-transparent"
              style={
                {
                  WebkitTextStroke: "0.4px #ffffff80",
                  WebkitTextFillColor: "transparent",
                  textShadow:
                    "-1px -1px 0 hsla(0,0%,100%,.2), 1px 1px 0 rgba(0,0,0,.1)",
                } as any
              } // silence the experimental webkit props
            >
              {title}
            </span>
            <span className="mx-8 text-center text-slate-300">{subtitle}</span>
          </h3>
        ))}
      </div>

      {/* TODO - List all other use cases with links */}
    </Container>
  );
}
