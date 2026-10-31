using System;
using System.Collections.Generic;

namespace Renki.Tasks
{
    public enum CookingDropMatch { Ignore, Accepted, Ambiguous }

    /// <summary>Correlates remote output; it is not a direct spawn witness.
    /// Pickup and the station/inventory receipts remain independently required.</summary>
    public sealed class CookingRemoteOutput
    {
        readonly string output;
        readonly long creator, producer;
        readonly bool recordCrafter;
        readonly int requested;
        readonly HashSet<string> before, accepted = new HashSet<string>();
        public int Count => accepted.Count;
        public bool Complete => Failure == null && Count == requested;
        public string Failure { get; private set; }

        public CookingRemoteOutput(string output, long creator, long producer, bool recordCrafter,
            int requested, IEnumerable<string> before)
        {
            if (string.IsNullOrEmpty(output) || creator == 0 || requested < 1 || requested > 10)
                throw new ArgumentException("invalid remote cooking removal");
            this.output = output; this.creator = creator; this.producer = producer;
            this.recordCrafter = recordCrafter; this.requested = requested;
            this.before = new HashSet<string>(before);
        }

        public CookingDropMatch Observe(string id, string prefab, long madeBy, double spawnedAfterRequestS,
            double distance, int stack, long crafter)
        {
            if (Failure != null) return CookingDropMatch.Ambiguous;
            if (string.IsNullOrEmpty(id) || before.Contains(id) || prefab != output || madeBy != creator ||
                !double.IsFinite(spawnedAfterRequestS) || spawnedAfterRequestS < -.1 || spawnedAfterRequestS > 3 ||
                !double.IsFinite(distance) || distance < 0 || distance > 2 || recordCrafter && crafter != producer)
                return CookingDropMatch.Ignore;
            if (stack != 1 || !accepted.Contains(id) && Count >= requested)
            {
                Failure = "ambiguous remote cooking output count or merged stack";
                return CookingDropMatch.Ambiguous;
            }
            return accepted.Add(id) ? CookingDropMatch.Accepted : CookingDropMatch.Ignore;
        }
    }
}
