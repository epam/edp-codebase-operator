package mock

type Closer struct {
	returnValue error
}

func (m *Closer) Close() error {
	return m.returnValue
}

func NewMockCloser(returnValue error) *Closer {
	return &Closer{returnValue: returnValue}
}
