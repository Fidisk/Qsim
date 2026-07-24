package components

import (
	glob "qsim/globals"
	"qsim/utils"
)

// Logical-bit fan-out.
//
// A LogicalBit can be linked to several hooks at once — e.g. one gate's
// output hook plus the input hooks of several consumers — so a single bit
// feeds a whole circuit without being detached from its source when it is
// connected elsewhere. The first link (HookID) is the primary one the bit
// anchors to while it has no other links; LinkedHookIDs holds the additional
// links. A bit may have at most one output-hook link.

// HasHook reports whether the bit is already linked to the given hook.
func (lb *LogicalBit) HasHook(id int32) bool {
	if id == 0 {
		return false
	}
	if lb.HookID == id {
		return true
	}
	for _, h := range lb.LinkedHookIDs {
		if h == id {
			return true
		}
	}
	return false
}

// AddHook links the bit to a hook, keeping the existing links (fan-out).
func (lb *LogicalBit) AddHook(id int32) {
	if id == 0 || lb.HasHook(id) {
		return
	}
	if lb.HookID == 0 {
		lb.HookID = id
		return
	}
	lb.LinkedHookIDs = append(lb.LinkedHookIDs, id)
}

// RemoveHook unlinks the given hook; the next link becomes the primary one.
func (lb *LogicalBit) RemoveHook(id int32) {
	if lb.HookID == id {
		lb.HookID = 0
		if len(lb.LinkedHookIDs) > 0 {
			lb.HookID = lb.LinkedHookIDs[0]
			lb.LinkedHookIDs = lb.LinkedHookIDs[1:]
		}
		return
	}
	for i, h := range lb.LinkedHookIDs {
		if h == id {
			lb.LinkedHookIDs = append(lb.LinkedHookIDs[:i], lb.LinkedHookIDs[i+1:]...)
			return
		}
	}
}

// LinkedHooks returns every hook the bit is currently linked to.
func (lb *LogicalBit) LinkedHooks() []*Hook {
	var hooks []*Hook
	if h, ok := utils.GetObjectFromID(lb.HookID).(*Hook); ok {
		hooks = append(hooks, h)
	}
	for _, id := range lb.LinkedHookIDs {
		if h, ok := utils.GetObjectFromID(id).(*Hook); ok {
			hooks = append(hooks, h)
		}
	}
	return hooks
}

// HasOutputLink reports whether any linked hook is a gate output, i.e. the
// bit is actively driven by a gate.
func (lb *LogicalBit) HasOutputLink() bool {
	for _, h := range lb.LinkedHooks() {
		if h.IsOutput {
			return true
		}
	}
	return false
}

// disconnectAllHooks unlinks every hook (detach mouse mode and Kill).
func (lb *LogicalBit) disconnectAllHooks() {
	for _, h := range lb.LinkedHooks() {
		h.IsHooked = false
		h.TargetID = 0
	}
	lb.HookID = 0
	lb.LinkedHookIDs = nil
	lb.SetWeight(glob.QubitDeterminatorWeight)
}
