package replicache

import "context"

type (
	Replicache[T any] struct {
		options  *Options
		mutators map[string]Mutator
	}

	Options struct {
		authFn AuthFn
	}

	AuthFn func(ctx context.Context, token string) bool
)

func New[T any](options ...Option) *Replicache[T] {
	r := new(Replicache[T])

	opts := &Options{
		authFn: func(ctx context.Context, token string) bool { return true },
	}
	for _, option := range options {
		option(opts)
	}
	r.options = opts

	return r
}

type Option func(o *Options)
type Mutator func(Mutation)

func WithAuth(fn func(ctx context.Context, token string) bool) Option {
	return func(o *Options) {
		o.authFn = fn
	}
}

func (r *Replicache[T]) Register(name string, mutator Mutator) error {
	if r.mutators == nil {
		r.mutators = make(map[string]Mutator)
	}

	if r.mutators[name] != nil {
		return ErrMutatorExists
	}

	r.mutators[name] = mutator
	return nil
}

func (r *Replicache[T]) Transact() {

}
