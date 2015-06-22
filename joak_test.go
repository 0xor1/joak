package joak

import(
	//`log`
	`time`
	`testing`
	`net/http`
	`appengine/aetest`
	`github.com/gorilla/mux`
	`google.golang.org/appengine`
	//`google.golang.org/appengine/datastore`
	`github.com/stretchr/testify/assert`
)

func Test_RouteLocalTest(t *testing.T){
	RouteLocalTest(mux.NewRouter(), nil, 300, `test`, &testEntity{}, nil, nil, nil)
}

func Test_RouteGaeProd(t *testing.T){
	c, _ := aetest.NewContext(nil)
	ctx := appengine.NewContext(c.Request().(*http.Request))
	dur1, _ := time.ParseDuration(`-1s`)

	err := RouteGaeProd(mux.NewRouter(), nil, 300, `test`, &testEntity{}, nil, nil, nil, dur1, dur1, ``, ctx, ``, ``, ``, ``)

	assert.Equal(t, `kind must not be an empty string`, err.Error(), `err should contain appropriate message`)

	err = RouteGaeProd(mux.NewRouter(), nil, 300, `test`, &testEntity{}, nil, nil, nil, dur1, dur1, `test`, ctx, ``, ``, ``, ``)

	assert.Equal(t, `deleteAfter must be a positive time.Duration`, err.Error(), `err should contain appropriate message`)

	dur2, _ := time.ParseDuration(`1s`)

	err = RouteGaeProd(mux.NewRouter(), nil, 300, `test`, &testEntity{}, nil, nil, nil, dur2, dur1, `test`, ctx, ``, ``, ``, ``)

	assert.Equal(t, `clearOutAfter must be a positive time.Duration`, err.Error(), `err should contain appropriate message`)

	err = RouteGaeProd(mux.NewRouter(), nil, 300, `test`, &testEntity{}, nil, nil, nil, dur2, dur2, `test`, ctx, ``, ``, ``, ``)

	assert.Nil(t, err, `err should contain appropriate message`)
}

type testEntity struct{
	Version 	int 		`datastore:",noindex"`
	DeleteAfter time.Time 	`datastore:""`
}

func (te *testEntity) GetVersion() int {
	return te.Version
}

func (te *testEntity) IncrementVersion() {
	te.Version++
}

func (te *testEntity) DecrementVersion() {
	te.Version--
}

func (te *testEntity) SetDeleteAfter(t time.Time) {
	te.DeleteAfter = t
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