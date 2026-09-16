package api

import "errors"

var (
	ErrInvalidStep           = errors.New("invalid step")
	ErrStepExists            = errors.New("step exists")
	ErrStepNotFound          = errors.New("step not found")
	ErrInvalidSpace          = errors.New("invalid space")
	ErrSpaceExists           = errors.New("space exists")
	ErrSpaceNotFound         = errors.New("space not found")
	ErrFlowExists            = errors.New("flow exists with a different plan")
	ErrFlowNotFound          = errors.New("flow not found")
	ErrGoalNotFound          = errors.New("goal step not found")
	ErrCircularDependency    = errors.New("circular dependency detected")
	ErrWorkItemNotFound      = errors.New("work item not found")
	ErrInvalidWorkTransition = errors.New("invalid work state transition")
)
