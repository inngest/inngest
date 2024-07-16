import Environment from './Environment';

export type NavProps = {
  collapsed: boolean;
};

export default function Navigation({ collapsed }: NavProps) {
  return (
    <div className="flex-start text-basis ml-5 mt-5 flex w-full flex-row items-center">
      <Environment envSlug="test" />
    </div>
  );
}
