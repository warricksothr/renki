using System;
using System.Collections.Generic;
using System.Linq;
using Renki.Protocol;

namespace Renki.Tasks
{
    public sealed class TransferResult
    {
        public readonly Dictionary<string, int> Moved = new Dictionary<string, int>();
        public readonly Dictionary<string, int> Remaining = new Dictionary<string, int>();
    }

    /// <summary>Count exact moves, including partial fits and protected stacks.
    /// The adapter must return only what the game actually transferred.</summary>
    public static class Transfer
    {
        // Alternate standing positions are bounded recovery for local conditional
        // cargo work, never a retry of permission, water or stopped-task refusals.
        public static bool CanReposition(bool matchExisting, V3 from, V3 target, Outcome failure) =>
            matchExisting && (target - from).Length <= 12 && failure != null &&
            (failure.Reason == Wire.ReasonUnreachable || failure.Reason == Wire.ReasonStuck);

        public static bool MatchesDestination(IReadOnlyList<TransferItem> requests, IEnumerable<string> contents)
        {
            var present = new HashSet<string>(contents, StringComparer.Ordinal);
            return requests.All(item => present.Contains(item.Prefab));
        }

        public static TransferResult Run<T>(IReadOnlyList<TransferItem> requests, IReadOnlyList<T> source,
            Func<T, string> prefab, Func<T, int> count, Func<T, bool> protect, Func<T, int, int> move)
        {
            var result = new TransferResult();
            foreach (var request in requests)
            {
                var matching = source.Where(it => prefab(it) == request.Prefab).ToArray();
                var wanted = request.Count == -1 ? matching.Sum(count) : request.Count;
                var left = wanted;
                foreach (var item in matching)
                {
                    if (left == 0) break;
                    if (protect(item)) continue;
                    var n = Math.Min(left, count(item));
                    var moved = move(item, n);
                    if (moved < 0 || moved > n) throw new InvalidOperationException("invalid transfer count");
                    left -= moved;
                }
                if (wanted != left) result.Moved[request.Prefab] = wanted - left;
                if (left > 0) result.Remaining[request.Prefab] = left;
            }
            return result;
        }
    }
}
