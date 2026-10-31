using System;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class DoorNavigationTests
    {
        [Fact]
        public void ClosedDoorCanGuideAPlanButCannotAuthorizeWalkingThroughIt()
        {
            var open = false;
            var refused = false;
            var key = true;
            var executing = false;
            V3? Probe(V3 from, V3 to)
            {
                if (Math.Abs(to.Z) > .1 || to.X < 0 || to.X > 4) return null;
                if (Math.Abs(to.X - 2) < .1 && !open &&
                    !DoorRules.MayIgnoreCollider(true, !open, key, refused, executing)) return null;
                return to;
            }
            LocalRoute Plan()
            {
                var plan = new LocalRoute(V3.Zero, new V3(4, 0, 0), Probe);
                while (plan.Status == RouteStatus.Searching) plan.Advance();
                return plan;
            }

            Assert.Equal(RouteStatus.Found, Plan().Status);
            executing = true;
            Assert.Null(Probe(new V3(1.5, 0, 0), new V3(2, 0, 0)));
            open = true;
            Assert.Equal(RouteStatus.Found, Plan().Status);
            open = false;
            executing = false;
            refused = true;
            Assert.Equal(RouteStatus.NoRoute, Plan().Status);
            refused = false;
            key = false;
            Assert.Equal(RouteStatus.NoRoute, Plan().Status);
        }

        [Fact]
        public void AnOpenDoorStillHasSolidFramesAndAnAnimatingLeaf()
        {
            Assert.False(DoorRules.MayIgnoreCollider(true, false, true, false, false));
            Assert.False(DoorRules.MayIgnoreCollider(false, true, true, false, false));
            Assert.False(DoorRules.Ahead(V3.Zero, new V3(1, 0, 0), new V3(1, 3, 0)));
            Assert.False(DoorRules.Ahead(V3.Zero, V3.Zero, new V3(.1, -3, 0)));
        }
    }
}
