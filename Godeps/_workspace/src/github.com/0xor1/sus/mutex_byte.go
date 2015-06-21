package sus

import(
	`sync`
)

type Marshaler func(src Version) ([]byte, error)
type Unmarshaler func(data []byte, dst Version) error
type ByteGetter func(id string) ([]byte, error)
type BytePutter func(id string, d []byte) error
type Deleter func(id string) error

// Creates and configures a store that stores entities by converting them to and from []byte and ensures versioning correctness with mutex locks.
func NewMutexByteStore(bg ByteGetter, bp BytePutter, d Deleter, m Marshaler, un Unmarshaler, idf IdFactory, vf VersionFactory, inee IsNonExtantError) Store {
	mtx := sync.Mutex{}

	getMulti := func(ids []string) ([]Version, error) {
		var err error
		var d []byte
		count := len(ids)
		vs := make([]Version, count, count)
		for i := 0; i < count; i++{
			d, err = bg(ids[i])
			if err != nil {
				break
			}
			vs[i] = vf()
			err = un(d, vs[i])
			if err != nil {
				break
			}
		}
		if err != nil {
			vs = nil
		}
		return vs, err
	}

	putMulti := func(ids []string, vs []Version) error {
		var err error
		var d []byte
		count := len(ids)
		for i := 0; i < count; i++{
			d, err = m(vs[i])
			if err != nil {
				break
			}
			err = bp(ids[i], d)
		}
		return err
	}

	delMulti := func(ids []string) (err error) {
		count := len(ids)
		for i := 0; i < count; i++ {
			err = d(ids[i])
			if err != nil {
				break
			}
		}
		return
	}

	rit := func(tran Transaction) error {
		mtx.Lock()
		defer mtx.Unlock()
		return tran()
	}

	return NewStore(getMulti, putMulti, delMulti, idf, vf, inee, rit)
}

type localEntityDoesNotExistError struct{
	id string
}

func (e localEntityDoesNotExistError) Error() string{
	return `entity with id "`+e.id+`" does not exist`
}
