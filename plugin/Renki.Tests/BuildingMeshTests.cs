using System;
using System.Linq;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class BuildingMeshTests
    {
        static V3[] Wall(double left, double right, double bottom = 0, double top = 3) => new[]
        {
            new V3(left, bottom, 0), new V3(right, bottom, 0), new V3(right, top, 0),
            new V3(left, bottom, 0), new V3(right, top, 0), new V3(left, top, 0),
        };

        [Fact]
        public void ActualFacesPreserveDoorOpeningAndFloorSeparation()
        {
            var doorway = new BuildingMesh(Wall(-2, -1).Concat(Wall(1, 2)).Concat(Wall(-1, 1, 2, 3)).ToArray());
            Assert.False(doorway.Blocks(new V3(0, 0, -2), new V3(0, 0, 2)));
            Assert.True(doorway.Blocks(new V3(1.5, 0, -2), new V3(1.5, 0, 2)));
            Assert.True(doorway.Blocks(new V3(.7, 0, -2), new V3(.7, 0, 2)));
            Assert.True(doorway.Blocks(new V3(0, 1, -2), new V3(0, 1, 2)));
            Assert.False(doorway.Blocks(new V3(0, 4, -2), new V3(0, 4, 2)));
            Assert.False(doorway.Blocks(new V3(1.5, 0, -.2), new V3(1.5, 0, -2)));
        }

        [Fact]
        public void RotatedCollisionFacesDoNotFillTheirWorldEnvelope()
        {
            V3 Rotate(V3 p) => new V3((p.X - p.Z) / Math.Sqrt(2), p.Y, (p.X + p.Z) / Math.Sqrt(2));
            var wall = new BuildingMesh(Wall(-2, 2).Select(Rotate).ToArray());
            Assert.True(wall.Blocks(Rotate(new V3(0, 0, -2)), Rotate(new V3(0, 0, 2))));
            Assert.False(wall.Blocks(Rotate(new V3(-1, 0, .5)), Rotate(new V3(1, 0, .5))));
        }

        [Fact]
        public void FloorsAndRampsBelowTheBodyAreSupportRatherThanSolidPrisms()
        {
            var floor = new BuildingMesh(new[] { new V3(-2, 0, -2), new V3(2, 0, -2), new V3(0, .3, 2) });
            Assert.False(floor.Blocks(new V3(0, 0, -1), new V3(0, .2, 1)));
            Assert.True(floor.Blocks(new V3(0, -1.1, -3), new V3(0, -1.1, 1)));
        }

        [Fact]
        public void MeasuredTrianglesAreCopiedAndInvalidMeshesAreRejected()
        {
            var faces = Wall(-1, 1); var wall = new BuildingMesh(faces);
            Array.Clear(faces, 0, faces.Length);
            Assert.True(wall.Blocks(new V3(0, 0, -2), new V3(0, 0, 2)));
            Assert.Throws<ArgumentException>(() => new BuildingMesh(new V3[BuildingMesh.TriangleLimit * 3 + 3]));
            Assert.Throws<ArgumentException>(() => new BuildingMesh(new[] { new V3(double.NaN, 0, 0), V3.Zero, V3.Forward }));
        }

        [Fact]
        public void SpatialEnvelopeSelectsFacesAndLocalSearchFindsTheOpening()
        {
            var doorway = new BuildingMesh(Wall(-3, -1).Concat(Wall(1, 3)).Concat(Wall(-1, 1, 2, 3)).ToArray());
            var index = new ObstacleIndex();
            index.Put(1, new ObstacleShape(new V3(0, 1.5, 0), 3, .01, 0, 3));
            V3? Probe(V3 a, V3 b) => index.Any(a, b, (_, from, to) => doorway.Blocks(from, to)) ? (V3?)null : b;
            var route = new LocalRoute(new V3(2, 0, -2), new V3(2, 0, 2), Probe, .25);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            Assert.Contains(route.Path, p => Math.Abs(p.X) <= .5 && Math.Abs(p.Z) < .5);
            for (var i = 1; i < route.Path.Count; i++) Assert.False(doorway.Blocks(route.Path[i - 1], route.Path[i]));
            // Removing a changed/destroyed part drops its broad-phase entry.
            var revision = index.Revision;
            index.Remove(1);
            Assert.True(index.Revision > revision);
            Assert.NotNull(Probe(new V3(2, 0, -1), new V3(2, 0, 1)));
        }
    }
}
