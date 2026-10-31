using System;
using System.Collections.Generic;
using System.Linq;

namespace Renki.Tasks
{
    public enum CookAction { Wait, Insert, Remove, Collect, Succeeded, Failed }

    public sealed class CookSlot
    {
        public string Item = "";
        public int Status;
        public long Producer;
    }

    /// <summary>A single input has one insertion and one removal attempt. Network
    /// uncertainty must not consume a second input or remove someone else's food.</summary>
    public sealed class CookingBatch
    {
        public static string Result(string raw) => raw == "RawMeat" ? "CookedMeat"
            : raw == "DeerMeat" ? "CookedDeerMeat" : raw == "NeckTail" ? "NeckTailGrilled" : null;
        public static bool Supported(string raw) => Result(raw) != null;
        readonly string raw, output;
        readonly long producer;
        readonly int rawBefore, outputBefore, maxYield;
        readonly double limit;
        int phase, slot = -1;
        double since;
        public string Failure { get; private set; }
        public int OutputCount { get; private set; }
        public int Slot => slot;
        public string Phase => phase == 0 ? "ready" : phase == 1 ? "inserting" : phase == 2 ? "cooking"
            : phase == 3 ? "removing" : phase == 4 ? "collecting" : phase == 5 ? "done" : "failed";

        public CookingBatch(string raw, long producer, int rawBefore, int outputBefore, int maxYield, double cookTime)
        {
            if (!Supported(raw) || rawBefore < 1 || outputBefore < 0 || maxYield < 1 || maxYield > 10 ||
                double.IsNaN(cookTime) || double.IsInfinity(cookTime) || cookTime < 1 || cookTime > 120)
                throw new ArgumentException("unsupported cooking batch");
            this.raw = raw; output = Result(raw); this.producer = producer;
            this.rawBefore = rawBefore; this.outputBefore = outputBefore; this.maxYield = maxYield;
            limit = cookTime * 2 + 10;
        }

        CookAction Fail(string why) { Failure = why; phase = -1; return CookAction.Failed; }

        public CookAction Next(double elapsed, int rawCount, int outputCount, IReadOnlyList<CookSlot> slots, bool heated, int spawned, int received = 0)
        {
            if (phase == -1) return CookAction.Failed;
            if (phase == 5) return CookAction.Succeeded;
            if (slots == null || slots.Count == 0 || slots.Count > 16 || slots.Any(s => s == null)) return Fail("cooking slots unavailable");
            if (double.IsNaN(elapsed) || double.IsInfinity(elapsed) || elapsed < 0 || elapsed > limit) return Fail("bounded cooking time elapsed");
            if (spawned < 0 || spawned > maxYield) return Fail("unexpected cooking yield");
            if (received < 0 || received > spawned) return Fail("unexpected cooking pickup receipt");
            var occupied = slots.Select((s, i) => (s, i)).Where(x => !string.IsNullOrEmpty(x.s.Item)).ToArray();
            if (phase == 0)
            {
                if (!heated) return Fail("the cooking station needs a working fire");
                if (occupied.Length != 0) return Fail("the cooking station is busy; existing food is not ours");
                if (rawCount != rawBefore) return Fail("the carried ingredient changed before insertion");
                phase = 1; since = elapsed; return CookAction.Insert;
            }
            if (phase == 1)
            {
                if (occupied.Length > 1 || occupied.Length == 1 &&
                    (occupied[0].s.Item != raw || occupied[0].s.Producer != producer || occupied[0].s.Status != 0))
                    return Fail("cooking input ownership is uncertain");
                if (rawCount == rawBefore - 1 && occupied.Length == 1)
                { slot = occupied[0].i; phase = 2; }
                else return elapsed - since <= 3 ? CookAction.Wait : Fail("input receipt unconfirmed; not inserting again");
            }
            if (phase == 2)
            {
                if (occupied.Length != 1 || occupied[0].i != slot || occupied[0].s.Producer != producer)
                    return Fail("the batch was removed or another cook changed the station");
                var current = occupied[0].s;
                if (current.Status == 2) return Fail("the cooking batch burnt");
                if (current.Status == 1 && current.Item == output)
                { phase = 3; since = elapsed; return CookAction.Remove; }
                if (current.Status != 0 || current.Item != raw) return Fail("unexpected cooking slot contents");
                if (!heated) return Fail("the cooking fire went out; input remains on the station");
                return CookAction.Wait;
            }
            if (phase == 3)
            {
                if (occupied.Any(x => x.i != slot || x.s.Producer != producer || x.s.Item != output || x.s.Status != 1))
                    return Fail("station changed during output removal");
                if (occupied.Length == 0 && spawned > 0)
                { OutputCount = spawned; phase = 4; since = elapsed; }
                else return elapsed - since <= 3 ? CookAction.Wait : Fail("output spawn receipt unconfirmed; not removing again");
            }
            if (phase == 4)
            {
                if (occupied.Length != 0 || spawned != OutputCount) return Fail("batch changed during collection");
                if (outputCount == outputBefore + OutputCount && received == OutputCount)
                { phase = 5; return CookAction.Succeeded; }
                if (outputCount > outputBefore + OutputCount) return Fail("unrelated food gain makes the receipt uncertain");
                return elapsed - since <= 15 ? CookAction.Collect : Fail("cooked output was not verified in inventory");
            }
            return Fail("invalid cooking phase");
        }
    }
}
