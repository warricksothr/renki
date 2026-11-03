using System;
using System.Collections.Generic;
using Newtonsoft.Json;

namespace Renki.Movement
{
    public sealed class TravelScope
    {
        public V3 Center;
        public double Radius, TravelM, DurationS;
        public bool Contains(V3 at) => Finite(at) && V3.FlatDistance(at, Center) <= Radius;
        public static bool Finite(V3 p) => Good(p.X) && Good(p.Y) && Good(p.Z);
        static bool Good(double n) => !double.IsNaN(n) && !double.IsInfinity(n) && Math.Abs(n) <= 1000000;
        public bool Valid => Finite(Center) && Good(Radius) && Radius >= 10 && Radius <= 250 &&
            Good(TravelM) && TravelM >= 1 && TravelM <= 500 && Good(DurationS) && DurationS >= 1 && DurationS <= 90;
    }

    /// <summary>Measured every native frame, including detours and failed legs.</summary>
    public sealed class BoundedTravel
    {
        readonly TravelScope scope;
        readonly double began;
        V3 last;
        readonly List<double[]> trail = new List<double[]>();
        public double Travel { get; private set; }
        public BoundedTravel(TravelScope scope, V3 at, double now)
        {
            this.scope = scope; began = now; last = at; trail.Add(at.ToArray());
        }
        public bool Clear(V3 from, V3 to) => scope.Contains(from) && scope.Contains(to);
        public string Observe(V3 at, double now)
        {
            if (!TravelScope.Finite(at)) return "invalid position";
            Travel += (at - last).Length; last = at;
            var previous = trail[trail.Count - 1];
            if (trail.Count < 128 && (at - new V3(previous[0], previous[1], previous[2])).Length >= 1) trail.Add(at.ToArray());
            if (!scope.Contains(at)) return "outside survey boundary";
            if (Travel >= scope.TravelM) return "survey leg travel limit";
            if (now - began >= scope.DurationS) return "survey leg time limit";
            return null;
        }
        public string Receipt(V3 at, string detail = null, string kind = "survey") => kind + ":" + JsonConvert.SerializeObject(new
        { travel = Travel, final = at.ToArray(), trail = trail.ToArray(), detail });
    }
}
