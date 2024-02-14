import styled from "@emotion/styled";

export default function DemoModal({
  show,
  onClickClose,
}: {
  show: boolean;
  onClickClose: () => void;
}) {
  return (
    !!show && (
      <Demo
        className="flex justify-center items-center"
        onClick={() => {
          onClickClose();
        }}
      >
        <div className="container aspect-video mx-auto max-w-2xl flex">
          <iframe
            src="https://www.youtube.com/embed/qVXzYBcJmGU?autoplay=1"
            title="Inngest Product Demo"
            frameBorder="0"
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
            allowFullScreen
            className="flex-1"
          ></iframe>
        </div>
      </Demo>
    )
  );
}

const Demo = styled.div`
  position: fixed;
  top: 0;
  z-index: 10;
  left: 0;
  width: 100%;
  max-width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.4);

  > div {
    box-shadow: 0 0 60px rgba(0, 0, 0, 0.5);
  }
`;
