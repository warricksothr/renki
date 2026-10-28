using System;

namespace Renki
{
    /// <summary>A world-space vector in metres, y up: Core's stand-in for
    /// UnityEngine.Vector3, so the maths can be tested without Unity.</summary>
    public readonly struct V3 : IEquatable<V3>
    {
        public readonly double X, Y, Z;

        public V3(double x, double y, double z)
        {
            X = x;
            Y = y;
            Z = z;
        }

        public static readonly V3 Zero = new V3(0, 0, 0);
        public static readonly V3 Forward = new V3(0, 0, 1);

        public static V3 operator +(V3 a, V3 b) => new V3(a.X + b.X, a.Y + b.Y, a.Z + b.Z);
        public static V3 operator -(V3 a, V3 b) => new V3(a.X - b.X, a.Y - b.Y, a.Z - b.Z);
        public static V3 operator *(V3 a, double k) => new V3(a.X * k, a.Y * k, a.Z * k);

        public double Length => Math.Sqrt(X * X + Y * Y + Z * Z);

        /// <summary>This vector with y set to 0.</summary>
        public V3 Flat => new V3(X, 0, Z);

        public V3 Normalized
        {
            get
            {
                var l = Length;
                return l < 1e-9 ? Zero : new V3(X / l, Y / l, Z / l);
            }
        }

        public static double Dot(V3 a, V3 b) => a.X * b.X + a.Y * b.Y + a.Z * b.Z;

        /// <summary>Distance on the ground plane, ignoring height.</summary>
        public static double FlatDistance(V3 a, V3 b) => (a - b).Flat.Length;

        public double[] ToArray() => new[] { Math.Round(X, 2), Math.Round(Y, 2), Math.Round(Z, 2) };

        public bool Equals(V3 o) => X == o.X && Y == o.Y && Z == o.Z;
        public override bool Equals(object obj) => obj is V3 o && Equals(o);
        public override int GetHashCode() => (X, Y, Z).GetHashCode();
        public override string ToString() => $"[{X:0.##}, {Y:0.##}, {Z:0.##}]";
    }
}
