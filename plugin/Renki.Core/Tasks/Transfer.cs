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
