package transform_test

//go:generate mockgen -source=interfaces_test.go -destination=mocks/transformfunc_interface.go -package=mocks

type TransformFunc interface {
	TransformFunc(interface{}) (interface{}, error)
}
