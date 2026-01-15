package engine

import (
	"context"

	"github.com/kode4food/ale"
	"github.com/kode4food/ale/data"
	"github.com/kode4food/ale/env"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const internalScriptLanguage = "internal"

func NewInternalAleEnv(engine *Engine, base *env.Environment) *AleEnv {
	snapshot := base.Snapshot()
	bindInternalFunctions(snapshot.GetRoot(), engine)
	return newAleEnv(snapshot)
}

func bindInternalFunctions(ns env.Namespace, engine *Engine) {
	_ = env.BindPublic(ns, "archive-flow", makeArchiveFlow(engine))
}

func makeArchiveFlow(engine *Engine) data.Procedure {
	return data.MakeProcedure(func(args ...ale.Value) ale.Value {
		flowID := api.FlowID(args[0].(data.String))
		if err := engine.archiveFlow(flowID); err != nil {
			panic(err)
		}
		if err := engine.raiseEngineEvent(
			context.Background(), api.EventTypeFlowArchived,
			api.FlowArchivedEvent{FlowID: flowID},
		); err != nil {
			panic(err)
		}
		return data.True
	}, 1)
}
