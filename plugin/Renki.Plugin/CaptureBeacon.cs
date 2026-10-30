using System;
using Renki.Protocol;
using UnityEngine;

namespace Renki.Plugin
{
    // Telemetry alone cannot prove capture. The adapter must also decode this
    // changing opt-in nonce from the exact OBS Game Capture source.
    sealed class CaptureBeacon
    {
        public CaptureProof Proof { get; private set; }
        double next;
        public void Update(bool active)
        {
            if (!active) { Proof = null; return; }
            if (Proof != null && Time.unscaledTimeAsDouble < next) return;
            Proof = new CaptureProof { Token = Guid.NewGuid().ToString("N").Substring(0, 16), At = DateTime.UtcNow };
            next = Time.unscaledTimeAsDouble + 0.5;
        }

        public void Draw()
        {
            if (Proof == null || UnityEngine.Event.current.type != EventType.Repaint) return;
            var matrix = GUI.matrix;
            var color = GUI.color;
            var depth = GUI.depth;
            try
            {
                GUI.matrix = Matrix4x4.Scale(new Vector3(Screen.width / 1280f, Screen.height / 720f, 1));
                GUI.depth = int.MinValue;
                var bits = "d39a" + Proof.Token;
                for (var i = 0; i < 80; i++)
                {
                    var nibble = System.Convert.ToInt32(bits[i / 4].ToString(), 16);
                    GUI.color = (nibble & (1 << (3 - i % 4))) != 0 ? Color.white : Color.black;
                    GUI.DrawTexture(new Rect(8 + i * 3, 4, 3, 8), Texture2D.whiteTexture);
                }
            }
            finally { GUI.matrix = matrix; GUI.color = color; GUI.depth = depth; }
        }
    }
}
