import { CommandSnippet } from "./CommandSnippet";

const SCRIPT = "curl -sfL https://cli.inngest.com/install.sh | sh";

const CLIInstall = () => <CommandSnippet command={SCRIPT} copy />;

export default CLIInstall;
