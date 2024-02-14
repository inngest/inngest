import React, { ReactElement } from "react";
import IconCalendar from "../Icons/Calendar";

type MetadataProps = {
  author: string;
  humanDate: string;
  readingTime: string;
};

export default function Metadata({
  author,
  humanDate,
  readingTime,
}: MetadataProps) {
  return (
    <p className="text-slate-300 text-sm mt-2 flex items-center gap-2">
      {!!author ? <>{author} &middot; </> : ""}
      <span className="flex items-center gap-1">
        <IconCalendar /> {humanDate}
      </span>{" "}
      &middot; <span>{readingTime}</span>
    </p>
  );
}
