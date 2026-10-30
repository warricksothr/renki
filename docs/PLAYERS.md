# Player profiles and presence

Tas's player-specific commands and routines share a permission resolver. A
profile grants a small set of capabilities; each command or routine checks the
capability it needs. Adding a friend to a greeting list never implicitly gives
that friend control of the bot.

| Profile | Commands | Locate Tas | Join greeting | Administration |
| --- | --- | --- | --- | --- |
| Existing owner | Yes | Yes | Yes | Existing owner/panel authority |
| Operator | Yes | Yes | Yes | No |
| Companion | No | Yes | Yes | No |
| Guest | No | Yes | No | No |
| Unassigned | No | Answers Tas's active Marco only | No | No |

Owners retain their authority even if an assignment names a lesser profile.
Operators can use the existing addressed chat commands, subject to the same
global task permissions, native safety and body ownership rules. Profiles do
not grant a panel login, change global permissions, or turn on combat. Assigning
profiles is an owner/panel operation; the agent cannot promote a player itself.
Names match the server's player names exactly, as existing ownership does. Chat
player IDs are retained with events and used to scope reply throttling; stable
account-based assignments can extend this principal model later.

## Panel controls

Open **Players & presence** to toggle server map-marker sharing, enable or
disable Marco/Polo, and assign/remove a player's profile. Owners already receive
the routines and do not need duplicate assignments. The normal settings API
persists these values, including changes made while the game is offline:

```json
{
  "players": {"profiles": {"Friend": "companion", "Pilot": "operator"}},
  "presence": {"sharePosition": true, "marcoPolo": true}
}
```

The setting keys are `players.profiles`, `presence.sharePosition` and
`presence.marcoPolo`. The default marker setting is off; Marco/Polo is on.
The panel distinguishes the desired marker setting from native observed state.
Marker changes use welcome/config messages and Valheim's ordinary
`SetPublicReferencePosition` setting on Unity's thread. They preserve current
movement, combat, automation leases and control revisions. The plugin's master
switch remains authoritative. While controlled, the configured marker setting
is maintained; reconnect/join reapplies the persisted choice.

## Marco/Polo exchanges

When a player with `greet` permission joins, Tas waits two seconds for their
character to load and for fresh perception. If he sees them, he stays quiet.
Otherwise he shouts **Marco!**. The initial roster when Tas himself joins is a
baseline, so logging the bot in does not greet everyone already on the server.
A fresh sighting means inclusion in local perception; this is not a raycast
claim about whether a wall blocks the view.

For two minutes after that Marco, **any player** can shout **Polo**. Tas replies
with the compass direction and horizontal distance **from that speaker to Tas**:
“Friend, travel about 100 m west to reach me, straight-line.” This limited reply
window follows the original request that anyone can answer Tas's call; it grants
no movement permission.

A player with `locate` permission can also shout **tas Marco** (or another
configured name/nickname). Tas shouts **Polo!** and describes the offset **from
himself to the caller**: “Friend, you're about 100 m east of me, straight-line.”
These two directions are intentionally opposite. Neither exchange asks Tas to
walk anywhere. The exact phrases bypass the language model. Guests and
companions still cannot issue follow, harvest or combat orders.

Replies use the position carried by the game's chat message and fresh bot
telemetry. Heights do not inflate the horizontal distance. Bearings use the
eight compass points and distances round to 5 m. Requests are bounded, scoped
to the current connection/world, and discarded across reconnect or world change.
Global/per-player throttles prevent repeated shouts from flooding chat. Missing
or stale observations never cause directions from an old world.

## Extending commands, routines and route knowledge

The initial capabilities are `command`, `locate`, and `greet`. Additional
routines should name the capability they need rather than introduce independent
player lists. Future profiles can split command permission into work, movement,
combat, and administration grants while retaining the global/native checks.
Per-player routine parameters can later add greeting preferences, delivery
locations or quiet periods beside the same principal/profile assignment.

Location replies currently describe a straight-line offset only. A later reply
can attach a route assessment with endpoint positions, world, age, navigation
revision and one of **unknown**, **checked route**, or **known obstruction**.
A cached route must be checked for the bot's traversal abilities and current
obstacle/door/water state before calling it usable. Observed terrain samples
can summarize biomes, water crossings and unknown coverage along that route;
terrain lying on the straight line is not evidence of a traversable path.
The response should distinguish remembered terrain from current observations
and leave uncovered segments unknown. A request to travel remains a separate,
authorized command or persistent goal.
