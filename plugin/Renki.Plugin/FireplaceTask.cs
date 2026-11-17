using System.Collections.Generic;
using System.Linq;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Refuel through Fireplace.UseItem, never toggle/claim ownership,
    /// edit inventory/ZDOs or offer an unrelated firework ingredient.</summary>
    sealed class FireplaceTask : IBodyTask
    {
        readonly string id;
        readonly AddToArgs args;
        readonly Fireplace fire;
        readonly Vector3 position;
        readonly float capacity;
        readonly string fuelName;
        readonly InteractionApproach approach;
        readonly List<Character> creatures = new List<Character>();
        readonly FireplaceFuel batch;
        readonly float began;
        float previousHp;
        bool cancelled;
        int traced;

        internal FireplaceTask(string id, AddToArgs args, Fireplace fire)
        {
            if (!Supported(fire, args.Item))
                throw new TaskRefused(Wire.ReasonInvalidTarget, "refillable finite fireplace and its exact fuel prefab are required");
            this.id = id; this.args = args; this.fire = fire;
            position = fire.transform.position; capacity = fire.m_maxFuel;
            fuelName = fire.m_fuelItem.m_itemData.m_shared.m_name;
            batch = new FireplaceFuel(args.Count, capacity);
            began = Time.unscaledTime; previousHp = Player.m_localPlayer.GetHealth();
            approach = new InteractionApproach("fuel", "fireplace", id, fire.gameObject,
                (me, feet) => SafeFooting(feet) && UseReach.From(me, fire.gameObject, CookingTask.InteractionRange(me), feet));
        }

        static bool Supported(Fireplace f, string item) => f != null && f.m_canRefill && !f.m_infiniteFuel &&
            f.m_nview != null && f.m_nview.IsValid() && f.m_fuelItem != null && f.m_fuelItem.name == item &&
            f.m_fuelItem.m_itemData?.m_shared != null && !string.IsNullOrEmpty(f.m_fuelItem.m_itemData.m_shared.m_name) && float.IsFinite(f.m_maxFuel) && f.m_maxFuel >= 1;

        bool SafeFooting(Vector3 feet)
        {
            var pieces = new List<Piece>(); Piece.GetAllPiecesInRadius(feet, 8, pieces);
            return CookingFire.SafeFooting(feet, pieces.Select(p => p.GetComponent<Fireplace>())
                .Where(f => f != null).Append(fire).Distinct());
        }

        int Count(Player me) => me.GetInventory().GetAllItems().Where(i => i.m_dropPrefab?.name == args.Item).Sum(i => i.m_stack);

        public Outcome Tick(double dt)
        {
            var me = Player.m_localPlayer;
            if (cancelled || me == null || me.IsDead()) return Finish(Outcome.Failed(Wire.ReasonStopped, "fire tending stopped"));
            if (!Supported(fire, args.Item) || Vector3.Distance(position, fire.transform.position) > .1f || fire.m_maxFuel != capacity ||
                fire.m_fuelItem.m_itemData.m_shared.m_name != fuelName || Perceiver.Id(fire.m_nview) != args.Entity)
                return Finish(Outcome.Failed(Wire.ReasonTargetLost, "fireplace or fuel definition changed"));
            if (me.NoCostCheat() || !PrivateArea.CheckAccess(position, 0f, false, false))
                return Finish(Outcome.Failed(Wire.ReasonNotAllowed, "ordinary fireplace access is required"));
            var hp = me.GetHealth();
            if (hp < Mathf.Min(previousHp, me.GetMaxHealth()) - .15f || hp < me.GetMaxHealth() * .8f ||
                me.GetFoods().Count == 0 || me.InWater() || me.IsSwimming())
                return Finish(Outcome.Failed(Wire.ReasonRefused, "unsafe health/food/water for fire tending"));
            previousHp = hp;
            creatures.Clear(); Character.GetCharactersInRange(me.transform.position, 12, creatures);
            if (creatures.Any(c => c != me && !c.IsDead() && !(c is Player) && !c.IsTamed() &&
                !(c.GetBaseAI() is AnimalAI) && BaseAI.IsEnemy(me, c)))
                return Finish(Outcome.Failed(Wire.ReasonRefused, "a hostile is too close to tend the fire"));
            // Pending receipts settle without movement or another native use.
            if (!SafeFooting(me.transform.position))
                return Finish(Outcome.Failed(Wire.ReasonRefused, "standing position has fire/smoke; tending stopped"));
            if (!batch.Pending)
            {
                var result = approach.Tick(dt, out var walking);
                if (result != null) return Finish(result);
                if (walking) return null;
            }
            Driver.Stand();
            if (!me.IsOnGround() || !SafeFooting(me.transform.position) ||
                !UseReach.Near(me, fire.gameObject, CookingTask.InteractionRange(me)))
                return Finish(Outcome.Failed(Wire.ReasonUnreachable, "safe standing sight/reach to the fireplace changed"));
            var fuel = fire.m_nview.GetZDO().GetFloat(ZDOVars.s_fuel);
            if (!batch.Pending && fuel > 0 && !fire.IsBurning())
                return Finish(batch.Added > 0 ? Outcome.Succeeded("fuel verified; fire is unlit, stopping further additions") :
                    Outcome.Failed(Wire.ReasonRefused, "fire already has fuel but is unlit; check ventilation/weather"));
            var pack = Count(me);
            var step = batch.Next(Time.unscaledTime - began, pack, fuel);
            if (batch.Added > traced)
            {
                traced = batch.Added;
                Trace("verified one-unit pack/fuel receipt", pack, fuel);
                RenkiPlugin.Instance.Send(InventoryReporter.Describe(me.GetInventory()));
            }
            if (step == FuelAction.Failed) return Finish(Outcome.Failed(pack < 1 && !batch.Pending ? Wire.ReasonInvalidTarget : Wire.ReasonRefused, batch.Detail));
            if (step == FuelAction.Ready) return Finish(Outcome.Succeeded(batch.Detail));
            if (step == FuelAction.Add)
            {
                var item = me.GetInventory().GetAllItems().FirstOrDefault(i => i.m_dropPrefab?.name == args.Item &&
                    i.m_stack > 0 && !i.m_equipped && i.m_shared.m_name == fire.m_fuelItem.m_itemData.m_shared.m_name &&
                    (Game.m_worldLevel <= 0 || i.m_worldLevel >= Game.m_worldLevel));
                if (item == null) return Finish(Outcome.Failed(Wire.ReasonInvalidTarget, "no ordinary carried fuel item is available"));
                Driver.LookAt = position; Driver.LookUntil = Time.time + 1;
                Trace("one native fuel use requested", pack, fuel);
                if (!fire.UseItem(me, item)) return Finish(Outcome.Failed(Wire.ReasonRefused, "fireplace refused fuel; not retrying"));
            }
            return null;
        }

        void Trace(string message, int pack, float fuel) => Tracer.Emit("fuel", message, new Dictionary<string, object>
        {
            ["id"] = id,
            ["fire"] = args.Entity,
            ["item"] = args.Item,
            ["added"] = batch.Added,
            ["requested"] = args.Count,
            ["pack"] = pack,
            ["fuel"] = fuel,
            ["lit"] = fire.IsBurning(),
            ["pos"] = Convert.ToArray(Player.m_localPlayer.transform.position)
        });
        Outcome Finish(Outcome result)
        {
            result.Detail = $"added {batch.Added}/{args.Count} {args.Item}; {result.Detail}";
            if (fire != null && fire.m_nview != null && fire.m_nview.IsValid())
                result.Detail += $"; fuel {fire.m_nview.GetZDO().GetFloat(ZDOVars.s_fuel):0.###}/{capacity}, lit={fire.IsBurning()}";
            Cancel(); return result;
        }
        public void Cancel()
        {
            if (cancelled) return;
            cancelled = true; approach.Cancel(); Driver.Stand();
            Tracer.Emit("fuel", "fire tending released; pending use is never replayed", new Dictionary<string, object>
            { ["id"] = id, ["fire"] = args.Entity, ["added"] = batch.Added, ["uncertain"] = batch.Pending });
        }
    }
}
