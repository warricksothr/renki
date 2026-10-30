// Original Renki calibration card. No game artwork, logos or remote resources.
export function Standby({ reason = "Tas is resting", phase = "standby" }) {
  return (
    <div
      class="standby-slate"
      role="status"
      aria-label={`Please stand by. ${reason}`}
    >
      <div class="slate-registration" aria-hidden="true">
        <i />
        <i />
        <i />
        <i />
      </div>
      <div class="slate-band">RENKI • FIELD SIGNAL SERVICE</div>
      <div class="slate-calibration" aria-hidden="true">
        <div class="slate-orbit">
          <span class="slate-mark">
            r<span>✦</span>
          </span>
        </div>
      </div>
      <div class="slate-message">
        <span class="slate-label">TRANSMISSION INTERMISSION</span>
        <strong>Please stand by</strong>
        <span class="slate-reason">{reason}</span>
      </div>
      <div class="slate-footer">
        <span>LOCAL SIGNAL • AUDIO OFF</span>
        <span class="slate-pulse" aria-hidden="true">
          ● ● ●
        </span>
        <span>
          {phase === "standby" ? "WE’LL BE RIGHT BACK" : "RESTORING PICTURE"}
        </span>
      </div>
    </div>
  );
}
