# Places and zones

Open **Places & zones** below the map. Select a saved marker or choose
**Add place**. Name it, choose its purpose, then use Tas's current position
or **Place on map**. Save to keep changes. A radius of zero is a point; a
positive radius is a circular zone, up to 100 m.

Map placement preserves the height field unless **Use terrain height** is
checked. This matters inside raised buildings: terrain under the floor is
not the floor. All three coordinates can be edited. **Go to saved place**
uses the last saved position through normal movement and safety checks.
Changing a zone's radius does not change movement's arrival distance.

Tas creates one standard place per purpose when he actually observes
evidence: storage chests, cooking equipment, trees, mineable stone/ore,
flint pickables or berry bushes. He does not invent unseen locations or
choose temporary drops/stumps as a renewable wood/flint site. Automatic
zones have a dashed outline. Editing or renaming one makes it owner-managed;
future discovery does not overwrite it. Deleting a purpose suppresses its
automatic recreation in that world. You can still add it manually.

**What's nearby?** lists up to 24 current or last-seen objects within the
zone (15 m around a point). Chests use the same remembered contents as Tas
and the hover view, with when they were checked. Unknown and observed-empty
contents differ. Coverage can be incomplete; an absent observation is not
proof that a resource is gone. Nearby means horizontal distance rather than
proof of access from the same floor.

Places live in the shared per-world memory and survive controller restarts.
Names normalize case, spaces/hyphens/underscores and leading "the" articles.
Names use letters and numbers, at most 64 characters. `home` remains the
dedicated home location. There are at most 128 places per world. Web edits
use the selected update time to reject stale changes rather than silently
overwrite another edit.

Owners can also say:

- `Tas, mark this as the mine` or `Tas, set waypoint mine`
- `Tas, move the mine here`
- `Tas, rename the mine to east quarry`
- `Tas, forget the east quarry`
- `Tas, go to the east quarry`
- `Tas, list waypoints`

Marking uses the owner's observed position. The panel's chat cannot mark an
unseen owner. Web coordinate edits use the coordinates explicitly entered.

The authenticated API is `GET /api/waypoints?world=…`, optionally with
`name=…` for nearby context, and `POST /api/waypoints` with `X-Renki-Act`:

```json
{"action":"set","world":"Midgard","name":"mine","expectedAt":"2026-10-30T00:00:00Z",
 "waypoint":{"name":"east quarry","kind":"stone","pos":[12,34,56],"radius":15}}
```

For a new place, leave `name` empty and omit `expectedAt`. Actions `delete`
and `go` take the saved name; deletion should include its `expectedAt`.
Movement requires Tas to be connected in the specified world. Editing memory
does not send a game action. API token holders may edit, as with the other
panel settings.

Direct controller `move_to` commands and the `move_to` behaviour template
accept `{"to":"east quarry"}`. The controller resolves the name in the
current world and sends coordinates to the plugin; colon-containing entity
IDs keep their existing meaning. A missing waypoint or a world change
during resolution sends no movement. Editing/deleting a place does not
redirect an already-dispatched move; the next order uses the new memory.

This supplies locations and context for later harvesting/hauling jobs.
Creating a zone does not itself install or start a harvesting behaviour.
