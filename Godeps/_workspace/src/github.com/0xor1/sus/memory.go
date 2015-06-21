package sus

import(
	`encoding/json`
)

func jsonMarshaler(v Version)([]byte, error){
	return json.Marshal(v)
}

func jsonUnmarshaler(d []byte, v Version) error{
	return json.Unmarshal(d, v)
}

// Creates and configures a store that stores entities by converting them to and from json []byte data and keeps them in the local system memory.
func NewJsonMemoryStore(idf IdFactory, vf VersionFactory) Store {
	return NewMemoryStore(jsonMarshaler, jsonUnmarshaler, idf, vf)
}

// Creates and configures a store that stores entities by converting them to and from []byte and keeps them in the local system memory.
func NewMemoryStore(m Marshaler, un Unmarshaler, idf IdFactory, vf VersionFactory) Store {
	store := map[string][]byte{}

	get := func(id string) ([]byte, error) {
		var err error
		d, exists := store[id]
		if !exists {
			err = localEntityDoesNotExistError{id}
		}
		return d, err
	}

	put := func(id string, d []byte) error {
		store[id] = d
		return nil
	}

	del := func(id string) error {
		delete(store, id)
		return nil
	}

	isNonExtantError := func(err error) bool {
		_, ok := err.(localEntityDoesNotExistError)
		return ok
	}

	return NewMutexByteStore(get, put, del, m, un, idf, vf, isNonExtantError)
}
