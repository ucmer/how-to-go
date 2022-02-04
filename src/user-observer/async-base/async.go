package async_base

import "context"

// Awaiter interface has the method signature for await
type Awaiter interface {
	Await() interface{}
}

type awaiter struct {
	await func(ctx context.Context) interface{}
}

func (function awaiter) Await() interface{} {
	return function.await(context.Background())
}

// Exec executes the async function
func Exec(function func() interface{}) Awaiter {
	var result interface{}
	chanel := make(chan struct{})

	go func() {
		defer close(chanel)
		result = function()
	}()

	return awaiter{
		await: func(ctx context.Context) interface{} {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-chanel:
				return result
			}
		},
	}
}
