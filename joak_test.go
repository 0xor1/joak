package joak

import(
	`time`
	`regexp`
	`testing`
	`github.com/0xor1/sus`
	`github.com/gorilla/mux`
	`github.com/stretchr/testify/assert`
)

func Test_RouteLocalTest(t *testing.T){
	RouteLocalTest(mux.NewRouter(), nil, 300, `test`, &testEntity{}, nil, nil ,nil)
}

func Test_RouteGaeProd(t *testing.T){
	dur, _ := time.ParseDuration(`10m`)
	err := RouteGaeProd(mux.NewRouter(), nil, 300, `test`, &testEntity{}, nil, nil ,nil, `testEntity`, dur, dur, `6455d34dy2e1cx47`, `54a1e479w2eb3z4b`, ``, ``)

	assert.Nil(t, err, `err should be nil`)

	err = RouteGaeProd(mux.NewRouter(), nil, 300, `test`, &testEntity{}, nil, nil ,nil, ``, dur, dur, `6455d34dy2e1cx47`, `54a1e479w2eb3z4b`, ``, ``)

	assert.Equal(t, `kind must not be an empty string`, err.Error(), `err should contain the appropriate message`)

	dur, _ = time.ParseDuration(`0m`)
	err = RouteGaeProd(mux.NewRouter(), nil, 300, `test`, &testEntity{}, nil, nil ,nil, `testEntity`, dur, dur, `6455d34dy2e1cx47`, `54a1e479w2eb3z4b`, ``, ``)

	assert.Equal(t, `deleteAfterDur must be a positive time.Duration`, err.Error(), `err should contain the appropriate message`)

	dur2, _ := time.ParseDuration(`10m`)
	err = RouteGaeProd(mux.NewRouter(), nil, 300, `test`, &testEntity{}, nil, nil ,nil, `testEntity`, dur2, dur, `6455d34dy2e1cx47`, `54a1e479w2eb3z4b`, ``, ``)

	assert.Equal(t, `clearOutDur must be a positive time.Duration`, err.Error(), `err should contain the appropriate message`)
}

func Test_Store(t *testing.T){
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	store := newMemoryStore(func()Entity{return &testEntity{Version:sus.NewVersion()}})

	id, e1, err := store.Create()

	assert.True(t, re.MatchString(id), `id is a valid uuid`)
	assert.Equal(t, 0, e1.GetVersion(), `entity starts with version equal to 0`)
	assert.Nil(t, err, `err should be nil`)

	err = store.Update(id, e1)

	e2, err := store.Read(id)
	assert.Equal(t, 1, e2.GetVersion(), `entity version is updated to 1`)
	assert.Nil(t, err, `err should be nil`)

	es, _ := store.(*entityStore)
	es.isForGae = true
	es.deleteAfter, _ = time.ParseDuration(`10m`)
	time := now().Add(es.deleteAfter)
	e, _ := e1.(Entity)
	gso := &gaeStoreObj{Entity:e}

	err = es.Update(id, gso)

	assert.Equal(t, 2, e.GetVersion(), `entity version is updated to 2`)
	assert.Equal(t, time, gso.DeleteAfter, `DeleteAfter should have been updated`)
	assert.Nil(t, err, `err should be nil`)
}

type testEntity struct{
	sus.Version
}

func (te *testEntity) IsActive() bool {
	return true
}

func (te *testEntity) CreatedBy() string {
	return `created_by`
}

func (te *testEntity) RegisterNewUser() (string, error) {
	return `new_user`, nil
}

func (te *testEntity) UnregisterUser(userId string) error {
	return nil
}

func (te *testEntity) Kick() bool {
	return false
}