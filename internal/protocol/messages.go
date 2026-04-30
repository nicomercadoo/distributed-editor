package protocol

type Operation int

const (
	Insert Operation = iota
	Delete
)

var operationToString = map[Operation]string{
	Insert: "I",
	Delete: "D",
}

var stringToOperation = func() map[string]Operation {
	m := make(map[string]Operation, len(operationToString))
	for k, v := range operationToString {
		m[v] = k
	}
	return m
}()

func (o Operation) String() string {
	if s, ok := operationToString[o]; ok {
		return s
	}
	return "unknown"
}

func ParseOperation(s string) (Operation, bool) {
	if v, ok := stringToOperation[s]; ok {
		return v, true
	}
	return 0, false
}

func (o Operation) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}

func (o *Operation) UnmarshalText(text []byte) error {
	if v, ok := ParseOperation(string(text)); ok {
		*o = v
		return nil
	}
	// default to Insert for backwards-compat; callers can validate separately
	*o = Insert
	return nil
}

// OperationMessage representa los comandos insert(c,p) y delete(p)
type OperationMessage struct {
    Op       Operation `json:"op"`
    Position int       `json:"pos"`
    Content  string    `json:"content,omitempty"` // solo para insert
}

// { "op": "I", "pos": 5, "content": "a" }
