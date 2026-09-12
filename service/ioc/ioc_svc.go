package service

import (
	"fmt"
	"reflect"
)

const (
	instanceIsNotPtr       = "ioc: 注入实例必须是指针"
	invalidTypeFormat      = "ioc: 无效类型(Name = %s, Type = %v)"
	notImplementsFormat    = "ioc: %v没有实现%v"
	notInterfaceTypeFormat = "ioc: 非接口类型(%v)"
)

// Container 依赖注入容器，可实例化，支持多容器隔离。
//
// 并发约定：Set / SetWithName 仅允许在启动初始化阶段调用；初始化完成后
// 容器进入只读，Get / Inject 可安全并发调用。运行时禁止再 Set（不加锁）。
type Container struct {
	values map[reflect.Type]map[string]reflect.Value
}

// New 创建新的注入容器。
func New() *Container {
	return &Container{
		values: make(map[reflect.Type]map[string]reflect.Value),
	}
}

// Get 根据接口类型与名称获取注入实例。
func (c *Container) Get(interfaceObj any, name string) any {
	return c.getValueWithName(interfaceObj, name).Interface()
}

// Inject 将依赖注入到 instance 中（instance 必须是指针）。
func (c *Container) Inject(instance any) {
	instanceValue := reflect.ValueOf(instance)
	if instanceValue.Kind() != reflect.Pointer {
		panic(instanceIsNotPtr)
	}

	c.inject(instanceValue)
}

// Set 以默认名称注册实例。
func (c *Container) Set(interfaceObj any, instance any) {
	c.SetWithName(interfaceObj, "", instance)
}

// SetWithName 以指定名称注册实例。
//
// 仅限启动初始化阶段调用（见 Container 的并发约定），运行时调用会导致数据竞争。
func (c *Container) SetWithName(interfaceObj any, name string, instance any) {
	interfaceType := getInterfaceType(interfaceObj)
	instanceType := reflect.TypeOf(instance)
	if !instanceType.Implements(interfaceType) {
		panic(
			fmt.Errorf(notImplementsFormat, instance, interfaceType),
		)
	}

	if _, ok := c.values[interfaceType]; !ok {
		c.values[interfaceType] = make(map[string]reflect.Value)
	}

	c.values[interfaceType][name] = reflect.ValueOf(instance)
}

func (c *Container) getValueWithName(interfaceObj any, name string) reflect.Value {
	interfaceType := getInterfaceType(interfaceObj)
	if values, ok := c.values[interfaceType]; ok {
		if v, ok := values[name]; ok {
			return v
		}
	}

	panic(
		fmt.Errorf(invalidTypeFormat, name, interfaceType),
	)
}

func (c *Container) inject(instanceValue reflect.Value) {
	if instanceValue.Kind() == reflect.Pointer {
		instanceValue = instanceValue.Elem()
	}

	instanceType := instanceValue.Type()
	for j := 0; j < instanceType.NumField(); j++ {
		field := instanceValue.Type().Field(j)
		fieldValue := instanceValue.FieldByIndex(field.Index)
		if field.Anonymous {
			if field.Type.Kind() == reflect.Struct {
				c.inject(fieldValue)
			}
			continue
		}

		name, ok := field.Tag.Lookup("inject")
		if !ok {
			return
		}

		if fieldValue.Kind() == reflect.Pointer {
			value := reflect.New(
				field.Type.Elem(),
			)
			fieldValue.Set(value)
			fieldValue = fieldValue.Elem()
		}

		v := c.getValueWithName(field.Type, name)
		fieldValue.Set(v)
	}
}

func getInterfaceType(interfaceObj any) reflect.Type {
	var interfaceType reflect.Type
	var ok bool
	if interfaceType, ok = interfaceObj.(reflect.Type); !ok {
		interfaceType = reflect.TypeOf(interfaceObj)
	}

	if interfaceType.Kind() == reflect.Pointer {
		interfaceType = interfaceType.Elem()
	}

	if interfaceType.Kind() != reflect.Interface {
		panic(
			fmt.Errorf(notInterfaceTypeFormat, interfaceType),
		)
	}

	return interfaceType
}
