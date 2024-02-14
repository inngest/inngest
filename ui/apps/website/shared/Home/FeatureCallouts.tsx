import Link from "next/link";
import Container from "../layout/Container";
import { ChevronRightIcon } from "@heroicons/react/20/solid";

const features = [
  {
    status: "New",
    title: "Branch environments on every platform",
    description:
      "Test your entire application end-to-end with an Inngest environment for every development branch that you deploy, without any extra work.",
    href: "/blog/branch-environments?ref=homepage-feature-callouts",
    icon: (
      <svg
        width="24"
        height="28"
        viewBox="0 0 24 28"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          fill-rule="evenodd"
          clip-rule="evenodd"
          d="M5.60002 9.21495C7.71824 8.7784 9.31084 6.9029 9.31084 4.65542C9.31084 2.0843 7.22654 0 4.65542 0C2.0843 0 0 2.0843 0 4.65542C0 6.83074 1.49198 8.65759 3.50846 9.16849V18.8315C1.49198 19.3424 0 21.1693 0 23.3446C0 25.9157 2.0843 28 4.65542 28C7.22654 28 9.31084 25.9157 9.31084 23.3446C9.31084 21.0971 7.71824 19.2216 5.60002 18.7851L5.60002 16.3827C7.69091 15.0899 10.2554 14.0927 14.1443 13.5515C16.1253 13.2758 17.2216 12.5957 18.1365 11.6124C18.8203 10.8776 19.2502 10.0297 19.5169 9.19649C21.5924 8.72772 23.1423 6.87258 23.1423 4.65542C23.1423 2.0843 21.058 0 18.4869 0C15.9157 0 13.8314 2.0843 13.8314 4.65542C13.8314 6.81351 15.2999 8.62863 17.2921 9.1561C17.1164 9.53154 16.8909 9.88067 16.6053 10.1876C16.0496 10.7849 15.3832 11.2 13.8559 11.4799C10.9433 11.7422 7.9796 12.6578 5.60002 13.9467V9.21495ZM6.88195 4.72289C6.88195 5.98982 5.8549 7.01687 4.58797 7.01687C3.32104 7.01687 2.294 5.98982 2.294 4.72289C2.294 3.45596 3.32104 2.42892 4.58797 2.42892C5.8549 2.42892 6.88195 3.45596 6.88195 4.72289ZM18.4191 7.01687C19.6861 7.01687 20.7131 5.98982 20.7131 4.72289C20.7131 3.45596 19.6861 2.42892 18.4191 2.42892C17.1522 2.42892 16.1252 3.45596 16.1252 4.72289C16.1252 5.98982 17.1522 7.01687 18.4191 7.01687ZM6.88195 23.4121C6.88195 24.679 5.8549 25.706 4.58797 25.706C3.32104 25.706 2.294 24.679 2.294 23.4121C2.294 22.1451 3.32104 21.1181 4.58797 21.1181C5.8549 21.1181 6.88195 22.1451 6.88195 23.4121Z"
          fill="#F5F5F5"
        />
      </svg>
    ),
  },
  {
    status: "Coming Soon",
    title: "Replay",
    description:
      "Never deal with the hassle of dead-letter-queues. Replay one or millions of failed functions at any time with the click of a button.",
    href: null,
    icon: (
      <svg
        width="21"
        height="24"
        viewBox="0 0 21 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          fill-rule="evenodd"
          clip-rule="evenodd"
          d="M18.565 14.1842C17.3588 18.686 12.7315 21.3576 8.22967 20.1513C6.74288 19.753 5.45859 18.9833 4.44636 17.9692L3.96836 17.4912L7.70173 17.4905C8.33726 17.4904 8.85237 16.9751 8.85226 16.3396C8.85215 15.704 8.33685 15.1889 7.70131 15.189L1.19052 15.1902C0.885316 15.1902 0.592638 15.3115 0.376869 15.5274C0.161101 15.7432 0.0399186 16.036 0.0399835 16.3412L0.0413516 22.8495C0.0414849 23.485 0.556798 24.0001 1.19234 24C1.82787 23.9999 2.34297 23.4846 2.34284 22.849L2.34205 19.1196L2.81747 19.5951C4.10788 20.8878 5.74605 21.8685 7.63401 22.3744C13.3636 23.9096 19.2528 20.5095 20.7881 14.7799C20.9526 14.166 20.5883 13.535 19.9744 13.3705C19.3605 13.206 18.7295 13.5703 18.565 14.1842ZM20.4506 8.47289C20.6664 8.25704 20.7876 7.9643 20.7876 7.65907L20.7865 1.15055C20.7864 0.515017 20.2711 -0.000103009 19.6356 1.54509e-08C19 0.000105961 18.4849 0.515394 18.485 1.15093L18.4856 4.88067L18.0103 4.40531C16.7199 3.11277 15.0814 2.13165 13.1936 1.62582C7.46401 0.0905889 1.57473 3.49077 0.0394962 9.22033C-0.124993 9.83421 0.239312 10.4652 0.853193 10.6297C1.46707 10.7942 2.09807 10.4299 2.26256 9.816C3.46881 5.3142 8.0961 2.64263 12.5979 3.84888C14.0848 4.24728 15.3691 5.01701 16.3813 6.03117L16.8587 6.50851L13.1272 6.50851C12.4917 6.50851 11.9765 7.02372 11.9765 7.65926C11.9765 8.29479 12.4917 8.81 13.1272 8.81H19.6368C19.942 8.81 20.2348 8.68873 20.4506 8.47289Z"
          fill="white"
        />
      </svg>
    ),
  },
];

export default function FeatureCallouts() {
  return (
    <Container className="-mt-36 tracking-tight">
      <div className="mx-auto max-w-6xl grid grid-cols-1 md:grid-cols-2 gap-5">
        {features.map(({ status, title, description, href, icon }, idx) => (
          <div
            key={idx}
            className="flex flex-col justify-between items-start py-8 px-10 rounded-lg"
            style={{
              backgroundImage:
                "radial-gradient(114.31% 100% at 50% 0%, #131E38 0%, #0A1223 100%)",
            }}
          >
            <span className="py-1 px-3 rounded-full text-teal-300 text-xs font-medium border border-teal-300/30 bg-teal-300/10 drop-shadow">
              {status}
            </span>
            <div className="mt-6 mb-3 flex flex-row gap-3">
              {icon}
              <h3 className="text-xl text-indigo-50 font-semibold">{title}</h3>
            </div>
            <p className="my-1.5 text-sm text-indigo-200 leading-relaxed">
              {description}
            </p>

            <Link
              href={href ? href : process.env.NEXT_PUBLIC_DISCORD_URL}
              className="group mt-6 flex flex-row items-center text-sm text-indigo-100 transition-all hover:text-indigo-50 whitespace-nowrap"
            >
              {href ? "Learn More" : "Join Our Community for First Access"}{" "}
              <ChevronRightIcon className="h-4 transition-all group-hover:translate-x-1" />
            </Link>
          </div>
        ))}
      </div>
    </Container>
  );
}
