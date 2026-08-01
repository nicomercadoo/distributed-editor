package client

type WSConn interface {
	WriteJSON(v interface{}) error
	ReadJSON(v interface{}) error
	Close() error
}
