using System;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class GroundCorridorTests
    {
        // Model the live probe's single-step reach. Long corridors must follow
        // changing support heights instead of probing from the original level.
        static Func<V3, V3, V3?> Terrain(Func<double, double, double> height) => (from, at) =>
        {
            var y = height(at.X, at.Z);
            return double.IsNaN(y) || Math.Abs(y - from.Y) > .6 ? (V3?)null : new V3(at.X, y, at.Z);
        };

        [Theory]
        [InlineData(.45)]
        [InlineData(-.45)]
        [InlineData(.577350269)]
        public void SmoothHillsideMayAccumulateSeveralMetresOfElevation(double grade)
        {
            var probe = Terrain((x, _) => x * grade);
            var end = new V3(6, 0, 0);
            Assert.Null(probe(V3.Zero, end)); // The old full-radius step probe.
            Assert.True(GroundCorridor.Check(V3.Zero, end, probe, out var reason), reason);
            Assert.Null(reason);
        }

        [Fact]
        public void FullDiameterAcceptsGradualChangesOfGrade()
        {
            var probe = Terrain((x, z) => .02 * x * x + .2 * z);
            var from = new V3(-6, .72, 0);
            Assert.True(GroundCorridor.Check(from, new V3(6, 0, 0), probe, out var reason), reason);
            Assert.True(GroundCorridor.Check(new V3(0, -1.2, -6), new V3(0, 0, 6), probe, out reason), reason);
        }

        [Theory]
        [InlineData(.65)]
        [InlineData(-.65)]
        public void SteepInclinationIsRejectedEvenWhenEachStepHasSupport(double grade)
        {
            Assert.False(GroundCorridor.Check(V3.Zero, new V3(6, 0, 0), Terrain((x, _) => x * grade), out var reason));
            Assert.Equal("grade", reason);
        }

        [Theory]
        [InlineData(1)]
        [InlineData(-1)]
        public void SharpCrestOrDipAtTheCenterIsRejectedBelowTheGradeLimit(double sign)
        {
            var probe = Terrain((x, _) => sign * Math.Abs(x) * .4);
            Assert.False(GroundCorridor.Check(new V3(-6, sign * 2.4, 0), new V3(6, 0, 0), probe, out var reason));
            Assert.Equal("sharp_change", reason);
        }

        [Fact]
        public void AbruptSmallLedgeCannotPassAsAnAcceptableAverageGrade()
        {
            var probe = Terrain((x, _) => x >= 3 ? .2 : 0);
            Assert.False(GroundCorridor.Check(V3.Zero, new V3(6, 0, 0), probe, out var reason));
            Assert.Equal("sharp_change", reason);
        }

        [Fact]
        public void BlockedUnsupportedOrWetIntermediateGroundIsNeverSkipped()
        {
            var calls = 0;
            V3? Probe(V3 from, V3 desired)
            {
                calls++;
                Assert.InRange(V3.FlatDistance(from, desired), 0, GroundCorridor.Spacing);
                return desired.X >= 2 && desired.X <= 3 ? (V3?)null : desired;
            }
            Assert.False(GroundCorridor.Check(V3.Zero, new V3(6, 0, 0), Probe, out var reason));
            Assert.Equal("blocked_or_unsupported", reason);
            Assert.InRange(calls, 2, 6);
        }

        [Fact]
        public void CharacterHeightOffsetDoesNotMakeADownhillDodgeLookTooSteep()
        {
            Assert.True(GroundCorridor.Check(new V3(0, .15, 0), new V3(5, 0, 0), Terrain((x, _) => -.5 * x), out var reason), reason);
        }

        [Fact]
        public void NonfiniteOrUnboundedInputNeverStartsPhysicsWork()
        {
            var calls = 0;
            V3? Probe(V3 _, V3 at) { calls++; return at; }
            foreach (var end in new[] { new V3(double.NaN, 0, 0), new V3(0, double.PositiveInfinity, 0), new V3(1000, 0, 0) })
                Assert.False(GroundCorridor.Check(V3.Zero, end, Probe, out _));
            Assert.Equal(0, calls);
            Assert.False(GroundCorridor.Check(V3.Zero, V3.Forward, (_, __) => new V3(0, double.NaN, 0), out _));
        }
    }
}
