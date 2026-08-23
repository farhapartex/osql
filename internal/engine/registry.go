package engine

import "slices"

type Registry struct {
	byVerb map[string]Executor
}

func NewRegistry(executors ...Executor) *Registry {
	r := &Registry{byVerb: make(map[string]Executor, len(executors))}
	for _, e := range executors {
		r.Register(e)
	}
	return r
}

func (r *Registry) Register(e Executor) {
	if r.byVerb == nil {
		r.byVerb = make(map[string]Executor)
	}
	r.byVerb[e.Verb()] = e
}

func (r *Registry) Lookup(verb string) (Executor, bool) {
	e, ok := r.byVerb[verb]
	return e, ok
}

func (r *Registry) Verbs() []string {
	verbs := make([]string, 0, len(r.byVerb))
	for v := range r.byVerb {
		verbs = append(verbs, v)
	}
	slices.Sort(verbs)
	return verbs
}
