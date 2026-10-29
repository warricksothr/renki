using System;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public class InteractionSightTests
    {
        [Fact]
        public void ThePlayersBodyDoesNotHideTheGrave()
        {
            Assert.True(InteractionSight.Visible(new[]
            {
                new SightHit(0.2, true, false),
                new SightHit(1.1, false, true),
            }));
        }

        [Fact]
        public void AWallOrFloorInFrontOfTheGraveStillBlocksIt()
        {
            Assert.False(InteractionSight.Visible(new[]
            {
                new SightHit(1.1, false, true),
                new SightHit(0.2, true, false),
                new SightHit(0.8, false, false),
            }));
        }

        [Fact]
        public void ABlockerBehindTheGraveDoesNotHideIt()
        {
            Assert.True(InteractionSight.Visible(new[]
            {
                new SightHit(2, false, false),
                new SightHit(1.1, false, true),
            }));
        }

        [Fact]
        public void NoTargetHitIsNotVisibility()
        {
            Assert.False(InteractionSight.Visible(Array.Empty<SightHit>()));
            Assert.False(InteractionSight.Visible(new[] { new SightHit(0.2, true, false) }));
        }

        [Fact]
        public void ABlockerAtTheSameDistanceWinsInEitherOrder()
        {
            var wall = new SightHit(1, false, false);
            var grave = new SightHit(1, false, true);
            Assert.False(InteractionSight.Visible(new[] { wall, grave }));
            Assert.False(InteractionSight.Visible(new[] { grave, wall }));
        }
    }
}
