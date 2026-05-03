package operation

type OperationType int

const (
	Insert OperationType = iota
	Delete
)

var operationToString = map[OperationType]string{
	Insert: "I",
	Delete: "D",
}

var stringToOperation = func() map[string]OperationType {
	m := make(map[string]OperationType, len(operationToString))
	for k, v := range operationToString {
		m[v] = k
	}
	return m
}()

func (o OperationType) String() string {
	if s, ok := operationToString[o]; ok {
		return s
	}
	return "unknown"
}

func ParseOperation(s string) (OperationType, bool) {
	if v, ok := stringToOperation[s]; ok {
		return v, true
	}
	return 0, false
}

func (o OperationType) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}

func (o *OperationType) UnmarshalText(text []byte) error {
	if v, ok := ParseOperation(string(text)); ok {
		*o = v
		return nil
	}
	// default to Insert for backwards-compat; callers can validate separately
	*o = Insert
	return nil
}
