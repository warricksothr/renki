using System;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class FightingGroundTests
    {
        static V3? Flat(V3 at) => new V3(at.X, 0, at.Z);

        [Fact]
        public void OuterTreeDoesNotExcludeAClearStanceAndAnotherEscapeDirection()
        {
            V3? Probe(V3 from, V3 at) => at.X > 3.5 ? (V3?)null : Flat(at);
            Assert.Null(FightingGround.Inspect(V3.Zero, Flat, Probe, out var region, out var escape));
            Assert.Null(region);
            Assert.InRange(escape.Value.X, -.001, .001);
            Assert.Equal(6, escape.Value.Z, 5);
        }

        [Fact]
        public void ClearCenterWithoutAnyWayOutIsRejected()
        {
            V3? Probe(V3 from, V3 at) => at.Flat.Length > 3.5 ? (V3?)null : Flat(at);
            var failure = FightingGround.Inspect(V3.Zero, Flat, Probe, out var region, out var escape);
            Assert.Equal("escape", region);
            Assert.Equal("blocked_or_unsupported", failure.Reason);
            Assert.Null(escape);
        }

        [Theory]
        [InlineData("unsupported")]
        [InlineData("steep")]
        [InlineData("crest")]
        [InlineData("dip")]
        public void UnsafeCentralGroundCannotBeRescuedByAnOpenOuterSector(string kind)
        {
            double Height(V3 at) => kind == "steep" ? at.X * .7 :
                kind == "crest" ? Math.Abs(at.X) * .4 : kind == "dip" ? -Math.Abs(at.X) * .4 : 0;
            V3? Project(V3 at) => new V3(at.X, Height(at), at.Z);
            V3? Probe(V3 from, V3 at) => kind == "unsupported" && Math.Abs(at.X) < .1 ? (V3?)null : Project(at);
            var failure = FightingGround.Inspect(V3.Zero, Project, Probe, out var region, out var escape);
            Assert.Equal("stance", region);
            Assert.Equal(kind == "unsupported" ? "blocked_or_unsupported" : kind == "steep" ? "grade" : "sharp_change", failure.Reason);
            Assert.Null(escape);
        }

        [Fact]
        public void SustainedGradualHillsideRemainsUsable()
        {
            V3? Project(V3 at) => new V3(at.X, .35 * at.X + .2 * at.Z, at.Z);
            V3? Probe(V3 from, V3 at)
            {
                var support = Project(at);
                Assert.InRange(Math.Abs(support.Value.Y - from.Y), 0, .6);
                return support;
            }
            Assert.Null(FightingGround.Inspect(V3.Zero, Project, Probe, out _, out var escape));
            Assert.Equal(2.1, escape.Value.Y, 5);
        }

        [Fact]
        public void EscapeCannotJumpAnUnsupportedStripEvenWithAClearEndpoint()
        {
            V3? Probe(V3 from, V3 at) => at.Flat.Length > 4 && at.Flat.Length < 5 ? (V3?)null : Flat(at);
            Assert.NotNull(FightingGround.Inspect(V3.Zero, Flat, Probe, out var region, out var escape));
            Assert.Equal("escape", region);
            Assert.Null(escape);
        }

        [Fact]
        public void SafeStanceCannotHideSteepExitsAndProbeWorkIsBounded()
        {
            var calls = 0;
            V3? Project(V3 at) => new V3(at.X, Math.Max(0, at.Flat.Length - 3) * .8, at.Z);
            V3? Probe(V3 from, V3 at) { calls++; return Project(at); }
            var failure = FightingGround.Inspect(V3.Zero, Project, Probe, out var region, out var escape);
            Assert.Equal("escape", region);
            Assert.Equal("grade", failure.Reason);
            Assert.Null(escape);
            Assert.InRange(calls, 1, 156); // At most twelve half-metre corridors, including support at their starts.
        }

        [Theory]
        [InlineData(true)]
        [InlineData(false)]
        public void MissingDryTerrainProjectionCannotBeAssumedSupported(bool stance)
        {
            V3? Project(V3 at) => stance || at.Flat.Length > 5 ? (V3?)null : Flat(at);
            var failure = FightingGround.Inspect(V3.Zero, Project, (_, at) => Flat(at), out var region, out var escape);
            Assert.Equal(stance ? "stance" : "escape", region);
            Assert.Equal("wet_or_unloaded_terrain", failure.Reason);
            Assert.Null(escape);
        }

        [Fact]
        public void RecheckDetectsNewCentralObstructionAndClearsTheOldEscape()
        {
            var blocked = false;
            V3? Probe(V3 from, V3 at) => blocked && at.Flat.Length < 1 ? (V3?)null : Flat(at);
            Assert.Null(FightingGround.Inspect(V3.Zero, Flat, Probe, out _, out var escape));
            Assert.NotNull(escape);
            blocked = true;
            Assert.NotNull(FightingGround.Inspect(V3.Zero, Flat, Probe, out var region, out escape));
            Assert.Equal("stance", region);
            Assert.Null(escape);
        }
    }
}
