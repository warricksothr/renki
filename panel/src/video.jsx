import { useEffect, useRef, useState } from "preact/hooks";
import { getStatus, subscribeStatus } from "./status.js";
import { VideoSession } from "./video.js";
import { Standby } from "./standby.jsx";

export function Video() {
  const [state, setState] = useState({
    phase: "connecting",
    reason: "Connecting to video",
  });
  const player = useRef();
  const session = useRef();
  useEffect(() => {
    const video = player.current;
    const controller = new VideoSession(setState, (stream) => {
      video.srcObject = stream;
      if (stream)
        video
          .play()
          .catch(() =>
            controller.show("unavailable", "Press retry to resume video"),
          );
    });
    session.current = controller;
    controller.setStatus(getStatus());
    const unsubscribe = subscribeStatus((s) => controller.setStatus(s));
    let disposed = false,
      callback,
      previousFrames = 0;
    const decoded = () => {
      if (disposed) return;
      controller.decoded();
      callback = video.requestVideoFrameCallback(decoded);
    };
    if (video.requestVideoFrameCallback)
      callback = video.requestVideoFrameCallback(decoded);
    const tick = () => {
      if (!video.requestVideoFrameCallback) {
        const frames = video.getVideoPlaybackQuality?.().totalVideoFrames || 0;
        if (frames > previousFrames) controller.decoded();
        previousFrames = frames;
      }
      controller.tick();
    };
    const timer = setInterval(tick, 1000);
    tick();
    return () => {
      disposed = true;
      clearInterval(timer);
      if (callback) video.cancelVideoFrameCallback(callback);
      unsubscribe();
      controller.dispose();
    };
  }, []);
  return (
    <div class="video-view">
      <div class="video-stage" data-video-state={state.phase}>
        <video
          ref={player}
          muted
          autoPlay
          playsInline
          aria-label="Tas game video"
          style={{
            visibility: state.phase === "gameplay" ? "visible" : "hidden",
          }}
        />
        {state.phase !== "gameplay" && (
          <Standby reason={state.reason} phase={state.phase} />
        )}
        {state.phase === "gameplay" && (
          <span class="video-live">GAME VIDEO • AUDIO OFF</span>
        )}
      </div>
      <div class="video-caption" role="status">
        <span>{state.reason}</span>
        <span>{state.phase}</span>
        {state.canRetry && (
          <button onClick={() => session.current.retry()}>Retry video</button>
        )}
      </div>
    </div>
  );
}
