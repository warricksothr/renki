namespace Renki.Perception
{
    /// <summary>
    /// Whether a building piece is a wall, from the box its colliders fill
    /// in the piece's own frame: tall, thin across and long along. A wall
    /// is drawn as the line along its foot (6.9). Floors are not tall,
    /// posts and beams are not long, roofs are not thin; chests and beds
    /// are reported as structures before this is asked.
    /// </summary>
    public static class WallShape
    {
        public const double MinHeight = 1.2, MaxThick = 1.05, MinLength = 0.9;

        /// <summary>The foot line, from A to B in the piece's frame, and the
        /// height; null when the box is not a wall's.</summary>
        public static (V3 A, V3 B, double H)? Of(V3 min, V3 max)
        {
            double ex = max.X - min.X, ey = max.Y - min.Y, ez = max.Z - min.Z;
            if (ey < MinHeight) return null;
            var alongX = ex >= ez;
            double length = alongX ? ex : ez, thick = alongX ? ez : ex;
            if (length < MinLength || length < 2 * thick || thick > MaxThick) return null;
            double cx = (min.X + max.X) / 2, cz = (min.Z + max.Z) / 2;
            return alongX
                ? (new V3(min.X, min.Y, cz), new V3(max.X, min.Y, cz), ey)
                : (new V3(cx, min.Y, min.Z), new V3(cx, min.Y, max.Z), ey);
        }
    }
}
