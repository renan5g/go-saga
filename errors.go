package saga

import "errors"

// Erros específicos do pacote saga
var (
	ErrSagaAlreadyExecuted = errors.New("saga has already been executed")
	ErrSagaCanceled        = errors.New("saga was canceled")
	ErrStepNotFound        = errors.New("step not found")
	ErrSagaCompensation    = errors.New("saga compensation failed")
)
