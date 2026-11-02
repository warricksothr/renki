def step(context, state, outcome):
    if context["objectiveSatisfied"]:
        return {"action": "done", "reason": "verified food readiness", "state": state}
    candidate = context.get("nextAction")
    if candidate == None:
        return {"action": "needs_plan", "reason": "allowed food actions exhausted", "state": state}
    return {"action": "run", "template": candidate["template"], "params": candidate["params"], "reason": "next bounded food preparation step", "state": state}
