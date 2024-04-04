package cqrs

import "reflect"

type Handler[TEvent, TResult any] interface {
	Handle(event TEvent) (TResult, error)
}

var handlers = make(map[reflect.Type]reflect.Value)

func Register[TEvent, TResult any](handler Handler[TEvent, TResult]) {
	eventType := reflect.TypeOf(handler.Handle).In(0)
	handlers[eventType] = reflect.ValueOf(handler)
}

func Handle[TResult, TEvent any](event TEvent) (TResult, error) {
	handler, ok := handlers[reflect.TypeOf(event)].Interface().(Handler[TEvent, TResult])

	if !ok {
		panic("handler is not a type of Handler[TEvent, TResult]")
	}

	return handler.Handle(event)
}
