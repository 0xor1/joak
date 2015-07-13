/*
Package sus provides data storage for entities that require sequential updates.
Any type of datastore can be created in the same manner as those available by default
in sus, Memory/File/AppEngine.
 */
package sus

import(
	`fmt`
)

// The interface that struct entities must include as anonymous fields in order to be used with sus stores.
type Version interface{
	GetVersion() int
	IncrementVersion()
	DecrementVersion()
}

// The core sus interface.
type Store interface{
	Create() (id string, v Version, err error)
	CreateMulti(count uint) (ids []string, vs []Version, err error)
	Read(id string) (v Version, err error)
	ReadMulti(ids []string) (vs []Version, err error)
	Update(id string, v Version) error
	UpdateMulti(ids []string, vs []Version) error
	Delete(id string) error
	DeleteMulti(ids []string) error
}

type IdFactory func() string
type VersionFactory func() Version
type RunInTransaction func(tran Transaction) error
type Transaction func() error
type GetMulti func(ids []string) ([]Version, error)
type PutMulti func(ids []string, vs []Version) error
type DeleteMulti func(ids []string) error
type IsNonExtantError func(error) bool
type EntityInitializer func(v Version) Version

// Create and configure a core store.
func NewStore(gm GetMulti, pm PutMulti, dm DeleteMulti, idf IdFactory, vf VersionFactory, ei EntityInitializer, inee IsNonExtantError, rit RunInTransaction) Store {
	return &store{gm, pm, dm, idf, vf, ei, inee, rit}
}

type store struct{
	getMulti			GetMulti
	putMulti			PutMulti
	deleteMulti			DeleteMulti
	idFactory 			IdFactory
	versionFactory 		VersionFactory
	entityInitializer 	EntityInitializer
	isNonExtantError	IsNonExtantError
	runInTransaction	RunInTransaction
}

// Creates a new versioned entity.
func (s *store) Create() (id string, v Version, err error) {
	ids, vs, err := s.CreateMulti(1)
	if len(ids) == 1 && len(vs) == 1 {
		id = ids[0]
		v = vs[0]
	}
	return
}

// Creates a set of new versioned entities.
func (s *store) CreateMulti(count uint) (ids []string, vs []Version, err error) {
	if count == 0 {
		return
	}
	icount := int(count)
	err = s.runInTransaction(func() error {
		ids = make([]string, count, count)
		vs = make([]Version, count, count)
		for i := 0; i < icount; i++ {
			ids[i] = s.idFactory()
			vs[i] = s.entityInitializer(s.versionFactory())
		}
		return s.putMulti(ids, vs)
	})
	return
}

// Fetches the versioned entity with id.
func (s *store) Read(id string) (v Version, err error) {
	vs, err := s.ReadMulti([]string{id})
	if len(vs) == 1 {
		v = vs[0]
	}
	return
}

// Fetches the versioned entities with id's.
func (s *store) ReadMulti(ids []string) (vs []Version, err error) {
	if len(ids) == 0 {
		return
	}
	err = s.runInTransaction(func() error {
		vs, err = s.getMulti(ids)
		if err != nil {
			if s.isNonExtantError(err) {
				err = &nonExtantError{err}
			}
		}
		return err
	})
	return
}

// Updates the versioned entity with id.
func (s *store) Update(id string, v Version) (err error) {
	err = s.UpdateMulti([]string{id}, []Version{v})
	return
}

// Updates the versioned entities with id's.
func (s *store) UpdateMulti(ids []string, vs []Version) (err error) {
	count := len(ids)
	if count != len(vs) {
		err = &idCountNotEqualToEntityCountError{count, len(vs)}
		return
	}
	if count == 0 {
		return
	}
	err = s.runInTransaction(func() error {
		oldVs, err := s.getMulti(ids)
		if err != nil {
			if s.isNonExtantError(err) {
				err = &nonExtantError{err}
			}
		} else {
			reverseI := 0
			for i := 0; i < count; i++ {
				if oldVs[i].GetVersion() != vs[i].GetVersion() {
					err = &nonsequentialUpdateError{ids[i]}
					reverseI = i
					break;
				}
				vs[i].IncrementVersion()
			}
			if err != nil {
				for i := 0; i < reverseI; i++ {
					vs[i].DecrementVersion()
				}
			} else {
				err = s.putMulti(ids, vs)
			}
		}
		return err
	})
	return
}

// Deletes the versioned entity with id.
func (s *store) Delete(id string) error {
	return s.DeleteMulti([]string{id})
}

// Deletes the versioned entities with id's.
func (s *store) DeleteMulti(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.runInTransaction(func() error {
		return s.deleteMulti(ids)
	})
}

type nonExtantError struct{
	inner error
}

func (e *nonExtantError) Error() string { return `Non extant error, inner error message: ` + e.inner.Error()}

type nonsequentialUpdateError struct{
	id string
}

func (e *nonsequentialUpdateError) Error() string { return `nonsequential update for entity with id "`+e.id+`"` }

type idCountNotEqualToEntityCountError struct{
	idCount int
	eCount	int
}

func (e *idCountNotEqualToEntityCountError) Error() string { return fmt.Sprintf(`id count (%d) not equal to entity count (%d)`, e.idCount, e.eCount) }
