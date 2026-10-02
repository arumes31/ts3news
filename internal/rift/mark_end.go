package rift

// MarkEnd records why a target disappeared without a finisher consuming its mark.
type MarkEnd struct {
	TargetName string `json:"target_name"`
	Reason     string `json:"reason"`
}

func (r *Run) endTargetMark(target Actor, reason string) {
	if r.Marked == "" || r.Marked != target.ID {
		return
	}
	name := target.Name
	if name == "" {
		name = "Target"
	}
	r.LastMarkEnd = &MarkEnd{TargetName: name, Reason: reason}
	r.Marked = ""
}
