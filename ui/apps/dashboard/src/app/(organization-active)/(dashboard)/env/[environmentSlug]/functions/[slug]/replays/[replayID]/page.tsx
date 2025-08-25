type Props = {
  params: {
    replayID: string;
  };
};

export default function Page({ params }: Props) {
  const replayID = decodeURIComponent(params.replayID);
  return <div>Replay ID: {replayID}</div>;
}
