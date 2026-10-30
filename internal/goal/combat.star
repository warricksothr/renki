def step(context, state, outcome):
    if not context["ready"]:
        return {"action": "wait", "waitS": 10, "reason": "recover health/stamina before another encounter", "state": state}
    if not context["target"]:
        return {"action": "wait", "waitS": 5, "reason": "wait for a fresh or alternate eligible Greyling", "state": state}
    return {"action": "run", "template": "combat", "params": context["params"], "reason": "try native encounter with a changed observation", "state": {"phase": "encounter"}}
