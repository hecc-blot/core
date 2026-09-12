package mocks

// MockContainer 是 IContainer 接口的 mock 实现，供单测复用。
type MockContainer struct{}

func (m *MockContainer) Get(interfaceObj interface{}, name string) interface{} { return nil }
func (m *MockContainer) Inject(instance interface{})                           {}
func (m *MockContainer) Set(interfaceObj interface{}, instance interface{})    {}
func (m *MockContainer) SetWithName(interfaceObj interface{}, name string, instance interface{}) {
}
