package store

import "errors"

// Errores propios del paquete store.
var (
	// ErrUnsupportedDriver indica que el driver no está en el registro de motores.
	ErrUnsupportedDriver = errors.New("motor no soportado")
	// ErrConnection indica un fallo al establecer la conexión.
	ErrConnection = errors.New("error de conexión")
	// ErrTableNotFound indica que la tabla solicitada no existe en el esquema.
	ErrTableNotFound = errors.New("tabla no encontrada")
)
