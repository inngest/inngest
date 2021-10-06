import {
  Activity,
  Clock,
  User,
  Database,
  GitMerge,
  Send,
  // Filter,
  // Zap,
} from "react-feather";
import Developers from "src/shared/Icons/Developers";

export const defaultIcon: React.FC = () => <Developers size={20} />;

export const categoryIcons: { [key: string]: React.FC } = {
  data: GitMerge,
  time: Clock,
  communication: Send,
  contacts: User,
  datastore: Database,
};

export const categoryIcon = (
  key: string | undefined
): React.FC<{ size: number }> => {
  return (key && categoryIcons[key]) || defaultIcon;
};
