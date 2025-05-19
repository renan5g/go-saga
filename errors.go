package saga

// Erros comuns
var (
	// ErrSagaCompensation é retornado quando ocorre falha na compensação
	ErrSagaCompensation = Error("saga compensation failed")
	// ErrSagaAlreadyExecuted é retornado quando tenta-se executar uma saga já executada
	ErrSagaAlreadyExecuted = Error("saga already executed")
	// ErrStepNotFound é retornado quando um passo não é encontrado
	ErrStepNotFound = Error("step not found")
	// ErrSagaCanceled é retornado quando a saga é cancelada através do contexto
	ErrSagaCanceled = Error("saga canceled from context")
)

// Error é um tipo de erro personalizado
type Error string

// Error implementa a interface error
func (e Error) Error() string {
	return string(e)
}
