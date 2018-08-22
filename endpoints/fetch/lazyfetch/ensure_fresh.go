package lazyfetch

import "reflect"

type EnsureFreshFunc func(fresh bool) (LazyFetchResponse, error)

func EnsureFresh(target interface{}, f EnsureFreshFunc) error {
	res, err := f(false)
	if err != nil {
		return err
	}

	v := reflect.ValueOf(res)
	stale := v.Elem().FieldByName("Stale").Bool()
	if stale {
		res, err = f(true)
		if err != nil {
			return err
		}
		v = reflect.ValueOf(res)
	}
	reflect.ValueOf(target).Elem().Set(v)
	return nil
}
