package gus

import(
	`fmt`
	`testing`
	`net/http`
	`github.com/0xor1/sus`
	`google.golang.org/appengine`
	`github.com/stretchr/testify/assert`
	`appengine/aetest`
)

func Test_GaeStore(t *testing.T){
	fgs := newFooGaeStore()

	id, foo, err := fgs.Create()

	assert.Equal(t, `1`, id, `id should be valid`)
	assert.Equal(t, 0, foo.GetVersion(), `Version should be initialised to 0`)
	assert.Nil(t, err, `err should be nil`)

	foo, err = fgs.Read(id)

	assert.Equal(t, 0, foo.GetVersion(), `Version should still be 0`)
	assert.Nil(t, err, `err should be nil`)

	err = fgs.Update(id, foo)

	assert.Equal(t, 1, foo.GetVersion(), `Version should be incremented to 1`)
	assert.Nil(t, err, `err should be nil`)

	foo, err = fgs.Read(id)

	assert.Equal(t, 1, foo.GetVersion(), `Version should still be 1`)
	assert.Nil(t, err, `err should be nil`)

	err = fgs.Delete(id)

	assert.Nil(t, err, `err should be nil`)

	foo, err = fgs.Read(id)

	assert.Equal(t, 0, foo.GetVersion(), `Version should be 0 as initialised`)
	assert.NotNil(t, err, `err should not be nil`)
}

type foo struct{
	Version int	`datastore:",noindex"`
}

func (f *foo) GetVersion() int {
	return f.Version
}

func (f *foo) IncrementVersion() {
	f.Version++
}

func (f *foo) DecrementVersion() {
	f.Version--
}

func newFooGaeStore() *fooGaeStore {
	c, _ := aetest.NewContext(nil)
	ctx := appengine.NewContext(c.Request().(*http.Request))
	idSrc := 0
	idf := func() string {
		idSrc++
		return fmt.Sprintf(`%d`, idSrc)
	}
	vf := func() sus.Version {
		return &foo{}
	}
	ei := func(v sus.Version) sus.Version {
		return v
	}
	return &fooGaeStore{
		inner: NewGaeStore(`foo`, ctx, idf, vf, ei),
	}
}

type fooGaeStore struct {
	inner sus.Store
}

func (fgs *fooGaeStore) Create() (id string, f *foo, err error) {
	id, v, err := fgs.inner.Create()
	if v != nil {
		f = v.(*foo)
	}
	return
}

func (fgs *fooGaeStore) CreateMulti(count uint) (ids []string, fs []*foo, err error) {
	ids, vs, err := fgs.inner.CreateMulti(count)
	if vs != nil {
		count := len(vs)
		fs = make([]*foo, count, count)
		for i := 0; i < count; i++ {
			fs[i] = vs[i].(*foo)
		}
	}
	return
}

func (fgs *fooGaeStore) Read(id string) (f *foo, err error) {
	v, err := fgs.inner.Read(id)
	if v != nil {
		f = v.(*foo)
	}
	return
}

func (fgs *fooGaeStore) ReadMulti(ids []string) (fs []*foo, err error) {
	vs, err := fgs.inner.ReadMulti(ids)
	if vs != nil {
		count := len(vs)
		fs = make([]*foo, count, count)
		for i := 0; i < count; i++ {
			fs[i] = vs[i].(*foo)
		}
	}
	return
}

func (fgs *fooGaeStore) Update(id string, f *foo) (err error) {
	return fgs.inner.Update(id, f)

}

func (fgs *fooGaeStore) UpdateMulti(ids []string, fs []*foo) (err error) {
	if fs != nil {
		count := len(fs)
		vs := make([]sus.Version, count, count)
		for i := 0; i < count; i++ {
			vs[i] = sus.Version(fs[i])
		}
		err = fgs.inner.UpdateMulti(ids, vs)
	}
	return
}

func (fgs *fooGaeStore) Delete(id string) (err error) {
	return fgs.inner.Delete(id)
}

func (fgs *fooGaeStore) DeleteMulti(ids []string) (err error) {
	return fgs.inner.DeleteMulti(ids)
}

