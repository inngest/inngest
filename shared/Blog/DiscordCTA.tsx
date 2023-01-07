import React from "react";
import styled from "@emotion/styled";

import { Button } from "../Button";
import DiscordLogo from "../Icons/Discord";

const DiscordCTA: React.FC<{ size?: "default" | "small" }> = ({
  size = "default",
}) => {
  return (
    <div className="max-w-[65ch] border-t-[2px] border-slate-800 pt-16 m-auto text-indigo-500">
      <DiscordLogo size={32} />
      <h2 className="text-white text-xl font-medium mt-6">
        Help shape the future of Inngest
      </h2>
      <p className="text-slate-400 mb-6 mt-2 text-sm">
        Ask questions, give feedback, and share feature requests
      </p>
      <Button variant="secondary" href={process.env.NEXT_PUBLIC_DISCORD_URL} arrow>
        Join our Discord!
      </Button>
    </div>
  );
};
export default DiscordCTA;
