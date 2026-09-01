package engine

import "context"

const ProgressInterval = 2048

type progressKey struct{}

func WithProgress(ctx context.Context, report func(scanned int)) context.Context {
	if report == nil {
		return ctx
	}
	return context.WithValue(ctx, progressKey{}, report)
}

func progressFrom(ctx context.Context) func(int) {
	report, _ := ctx.Value(progressKey{}).(func(int))
	return report
}
