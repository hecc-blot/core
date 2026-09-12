package ioc

// IContainer 依赖注入容器接口。
// 框架组件（api/sse）依赖此接口而非具体实现，业务方可替换为自己的容器实现。
type IContainer interface {
	// Get 根据接口类型与名称获取注入实例。
	Get(interfaceObj interface{}, name string) interface{}
	// Inject 将依赖注入到 instance 中（instance 必须是指针）。
	Inject(instance interface{})
	// Set 以默认名称注册实例。
	Set(interfaceObj interface{}, instance interface{})
	// SetWithName 以指定名称注册实例。
	SetWithName(interfaceObj interface{}, name string, instance interface{})
}
