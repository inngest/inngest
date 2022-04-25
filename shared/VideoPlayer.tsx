import React, { useEffect, useRef, useState } from "react";
import styled from "@emotion/styled";

type VideoPlayerProps = {
  className: string;
  src: string;
  autoPlay?: boolean;
  muted?: boolean;
  duration: number;
  chapters?: {
    name: string;
    start: number;
  }[];
};

const VideoPlayer: React.FC<VideoPlayerProps> = ({
  className = "",
  src,
  autoPlay = true,
  muted = true,
  duration,
  chapters = [{ start: 0 }],
}) => {
  const video = useRef<HTMLVideoElement>();
  const [progressPercentage, setProgressPercentage] = useState(0);

  // Autoplay the video when the video visible
  useEffect(() => {
    if (!autoPlay) {
      return;
    }
    const observer = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting) {
        video.current.play();
      } else {
        video.current.pause();
      }
    });
    observer.observe(video.current);
  }, [video.current]);

  const onTimeUpdate = (e: React.SyntheticEvent) => {
    const percentCompleted =
      (video.current.currentTime / video.current.duration) * 100;
    setProgressPercentage(percentCompleted);
  };
  const onVideoClick = (e: React.MouseEvent) => {
    if (video.current.paused || video.current.ended) {
      video.current.play();
    } else {
      video.current.pause();
    }
  };
  const jumpTo = (location: number) => {
    video.current.currentTime = location;
    video.current.play();
  };
  return (
    <Player className={className}>
      <video
        ref={video}
        onTimeUpdate={onTimeUpdate}
        onClick={onVideoClick}
        controls={false}
        autoPlay={autoPlay}
        muted={muted}
      >
        <source src={src} type="video/mp4" />
      </video>
      <Controls>
        {/* <Progress style={{ width: `${progress}%` }} /> */}
        {chapters.map(({ name, start }, idx) => {
          // let location = duration ? (start / duration) * 100 : -100;
          const startPercentage = (start / duration) * 100;
          const endPercentage = chapters[idx + 1]?.start
            ? (chapters[idx + 1].start / duration) * 100
            : 100;
          const completion =
            progressPercentage > endPercentage
              ? 100
              : progressPercentage < startPercentage
              ? 0
              : ((progressPercentage - startPercentage) /
                  (endPercentage - startPercentage)) *
                100;
          const isActive = completion > 0 && completion < 100;
          return (
            <Chapter
              key={`chapter-${idx}`}
              startPercentage={startPercentage}
              endPercentage={endPercentage}
              isActive={isActive}
              onClick={() => jumpTo(start)}
            >
              <ProgressBar>
                <Progress completion={completion} />
              </ProgressBar>
              <ChapterName key={idx}>{name}</ChapterName>
            </Chapter>
          );
        })}
      </Controls>
    </Player>
  );
};

const Player = styled.div`
  position: relative;
  cursor: pointer;
  background: #1e1f22;
  padding: 0 0 40px 0;
  box-shadow: 0px 20px 350px rgba(70, 54, 245, 0.12),
    0px 10px 30px rgba(0, 0, 0, 0.85);
`;
const Controls = styled.div`
  position: absolute;
  height: 6px;
  bottom: 8px;
  left: 8px;
  right: 8px;
`;
const Chapter = styled.div<{
  startPercentage: number;
  endPercentage: number;
  isActive: boolean;
}>`
  position: absolute;
  top: 0px;
  left: ${({ startPercentage }) => `${startPercentage}%`};
  width: calc(
    ${({ startPercentage, endPercentage }) =>
        `${endPercentage - startPercentage}%`} - 3px
  );
  height: 100%;
  background-color: var(--stroke-color);
  transition: all 0.2s ease-in-out;
  &:hover {
    // Only highlight if it's clickable to jump to the chapter
    background-color: ${({ isActive }) =>
      isActive ? "var(--stroke-color)" : "var(--stroke-color-light)"};
  }
`;
const ChapterName = styled.span`
  position: absolute;
  top: -20px;
  left: 0px;
  font-size: 12px;
  text-transform: uppercase;
  color: #fff;
  text-shadow: 0 0 2px rgba(0, 0, 0, 0.5);
`;
const ProgressBar = styled.div`
  position: absolute;
  height: 100%;
  width: 100%;
`;
const Progress = styled.div<{ completion: number }>`
  position: absolute;
  height: 100%;
  width: ${({ completion }) => `${completion}%`};
  background-color: var(--primary-color);
  // NOTE - Animation disabled for now as skipping to chapters makes it look weird

  // When clicking on previous chapter, we want to hide the progress
  // instantly, not animate it backwards
  /* visibility: ${({ completion }) =>
    completion === 0 ? "hidden" : "block"}; */
  /** Chrome seems to fire the onTimeUpdate event every 250ms
    * so we use that to try to keep the progress bar smooth */
  /* transition: width 250ms linear; */
`;

export default VideoPlayer;
