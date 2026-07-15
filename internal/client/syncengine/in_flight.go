package syncengine

import "proyecto-final-redes/internal/protocol/operation"

type InFlightOperation struct {
	op       operation.Operation
	attempts int
	received bool
}

func NewInFlightOperation(op operation.Operation) *InFlightOperation {
	return &InFlightOperation{
		op:       op,
		attempts: 3,
		received: false,
	}
}

func (i *InFlightOperation) Op() operation.Operation {
	return i.op
}

func (i *InFlightOperation) SetOp(op operation.Operation) {
	i.op = op
}

func (i *InFlightOperation) Attempts() int {
	return i.attempts
}

func (i *InFlightOperation) DecrementAttempt() {
	if i.attempts > 0 {
		i.attempts--
	}
}

func (i *InFlightOperation) IsReceived() bool {
	return i.received
}

func (i *InFlightOperation) SetReceived(received bool) {
	i.received = received
}
