package oak

import(
	`bytes`
	`errors`
	`testing`
	`net/http`
	js `encoding/json`
	`net/http/httptest`
	`github.com/gorilla/mux`
	`github.com/gorilla/sessions`
	`github.com/stretchr/testify/assert`
)

func Test_create_without_existing_session(t *testing.T){
	w, r := setup(nil, nil, nil, _CREATE, ``)

	tr.ServeHTTP(w, r)

	resp := Json{}
	readTestJson(w, &resp)
	assert.Equal(t, `test_entity_id`, resp[_ID].(string), `response json should contain the returned entityId`)
	assert.Equal(t, `test_creator_user_id`, tss.session.Values[_USER_ID], `session should have the provided user id`)
	assert.Equal(t, resp[_ID].(string), tss.session.Values[_ENTITY_ID].(string), `session should have a entityId matching the json response`)
	assert.Equal(t, tes.entity, tss.session.Values[_ENTITY].(*testEntity), `session should have the entity`)
}

func Test_create_with_existing_session(t *testing.T){
	w, r := setup(nil, nil, nil, _CREATE, ``)
	tes.Create()
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	resp := Json{}
	readTestJson(w, &resp)
	assert.Equal(t, `test_pre_set_entity_id`, resp[_ID].(string), `response json should have the existing entityId`)
	assert.Equal(t, `test_pre_set_user_id`, tss.session.Values[_USER_ID], `session should have the existing user id`)
	assert.Equal(t, resp[_ID].(string), tss.session.Values[_ENTITY_ID].(string), `session should have a entityId matching the json response`)
	assert.Equal(t, entity, tss.session.Values[_ENTITY], `session should have the existing entity`)
}

func Test_create_with_store_error(t *testing.T){
	w, r := setup(nil, nil, nil, _CREATE, ``)
	tes.createErr = errors.New(`test_create_error`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_create_error\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session.Values[_USER_ID], `session should not have a userId`)
	assert.Nil(t, tss.session.Values[_ENTITY_ID], `session should not have an entityId`)
	assert.Nil(t, tss.session.Values[_ENTITY], `session should not have an entity`)
}

func Test_join_without_existing_session(t *testing.T){
	w, r := setup(func(userId string, e Entity)Json{return Json{"test": "yo"}}, nil, nil, _JOIN, `{"`+_ID+`":"req_test_entity_id"}`)
	tes.Create()

	tr.ServeHTTP(w, r)

	resp := Json{}
	readTestJson(w, &resp)
	assert.Equal(t, `yo`, resp[`test`].(string), `response json should contain the returned data from getJoinResp`)
	assert.Equal(t, 0, int(resp[_VERSION].(float64)), `response json should contain the version number`)
	assert.Equal(t, `test_user_id`, tss.session.Values[_USER_ID], `session should have the provided user id`)
	assert.Equal(t, `req_test_entity_id`, tss.session.Values[_ENTITY_ID].(string), `session should have the entityId`)
	assert.Equal(t, tes.entity, tss.session.Values[_ENTITY].(*testEntity), `session should have the entity`)
}

func Test_join_with_existing_session(t *testing.T){
	w, r := setup(func(userId string, e Entity)Json{return Json{"test": "yo"}}, nil, nil, _JOIN, `{"`+_ID+`":"req_test_entity_id"}`)
	tes.Create()
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	resp := Json{}
	readTestJson(w, &resp)
	assert.Equal(t, `yo`, resp[`test`].(string), `response json should contain the returned data from getJoinResp`)
	assert.Equal(t, 0, int(resp[_VERSION].(float64)), `response json should contain the version number`)
	assert.Equal(t, `test_pre_set_user_id`, tss.session.Values[_USER_ID], `session should have the existing user id`)
	assert.Equal(t, `test_pre_set_entity_id`, tss.session.Values[_ENTITY_ID], `response json should have the existing entityId`)
	assert.Equal(t, entity, tss.session.Values[_ENTITY], `session should have the existing entity`)
}

func Test_join_with_request_missing_id(t *testing.T) {
	w, r := setup(nil, nil, nil, _JOIN, `{}`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, _ID + " value must be included in request\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_join_with_request_nonstring_id(t *testing.T) {
	w, r := setup(nil, nil, nil, _JOIN, `{"`+_ID+`": true}`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, _ID + " must be a string value\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_join_with_entity_store_read_error(t *testing.T) {
	w, r := setup(nil, nil, nil, _JOIN, `{"`+_ID+`": "test_entity_id"}`)
	tes.readErr = errors.New(`test_read_error`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_read_error\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_join_with_entity_store_update_error_on_second_update_pass(t *testing.T) {
	w, r := setup(nil, nil, nil, _JOIN, `{"`+_ID+`": "test_entity_id"}`)
	callCount := 0
	tes.update = func(entityId string, entity Entity) error{
		if callCount == 0 {
			callCount++
			return errors.New(`nonsequential update for entity with id "test_entity_id"`)
		} else {
			return errors.New(`test_update_error`)
		}
	}
	tes.updateErr = errors.New(`test_update_error`)
	tes.entity = &testEntity{kick:func()bool{return true}}

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_update_error\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_join_with_never_ending_nonsequential_update_errors(t *testing.T) {
	w, r := setup(nil, nil, nil, _JOIN, `{"`+_ID+`": "test_entity_id"}`)
	callCount := 0
	tes.update = func(entityId string, entity Entity) error{
		callCount++
		return errors.New(`nonsequential update for entity with id "test_entity_id"`)
	}
	tes.entity = &testEntity{kick:func()bool{return true}}

	tr.ServeHTTP(w, r)

	assert.Equal(t, "nonsequential update for entity with id \"test_entity_id\"\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_poll_with_no_change(t *testing.T) {
	w, r := setup(nil, nil, nil, _POLL, `{"`+_ID+`": "test_entity_id", "`+_VERSION+`": 0}`)
	tes.Create()

	tr.ServeHTTP(w, r)

	assert.Equal(t, `{}`, w.Body.String(), `response body should be error message`)
	assert.Equal(t, 200, w.Code, `return code should be 200`)
}

func Test_poll_with_change(t *testing.T) {
	w, r := setup(nil, func(userId string, entity Entity)Json{return Json{"test": "yo"}}, nil, _POLL, `{"`+_ID+`": "test_entity_id", "`+_VERSION+`": -1}`)
	tes.Create()

	tr.ServeHTTP(w, r)

	resp := Json{}
	readTestJson(w, &resp)
	assert.Equal(t, `yo`, resp[`test`].(string), `response json should contain the returned data from getJoinResp`)
	assert.Equal(t, 0, int(resp[_VERSION].(float64)), `response json should contain the version number`)
}

func Test_poll_with_session_user_and_entity_is_active(t *testing.T) {
	w, r := setup(nil, func(userId string, entity Entity)Json{return Json{"test": "yo"}}, nil, _POLL, `{"`+_ID+`": "test_entity_id", "`+_VERSION+`": -1}`)
	tes.Create()
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_entity_id`
	entity := &testEntity{kick:func()bool{return true}}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	resp := Json{}
	readTestJson(w, &resp)
	assert.Equal(t, `yo`, resp[`test`].(string), `response json should contain the returned data from getJoinResp`)
	assert.Equal(t, 0, int(resp[_VERSION].(float64)), `response json should contain the version number`)
	assert.Equal(t, `test_pre_set_user_id`, s.Values[_USER_ID], `session userId should be unchanged`)
	assert.Equal(t, `test_entity_id`, s.Values[_ENTITY_ID], `session entityId should be unchanged`)
	assert.NotEqual(t, entity, s.Values[_ENTITY], `session entity should not be it's original value`)
	assert.Equal(t, tes.entity, s.Values[_ENTITY], `session entity should be updated to the stores entity`)
}

func Test_poll_with_session_user_and_entity_is_not_active(t *testing.T) {
	w, r := setup(nil, func(userId string, entity Entity)Json{return Json{"test": "yo"}}, nil, _POLL, `{"`+_ID+`": "test_entity_id", "`+_VERSION+`": -1}`)
	tes.Create()
	tes.entity.isActive = func()bool{return false}
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	resp := Json{}
	readTestJson(w, &resp)
	assert.Equal(t, `yo`, resp[`test`].(string), `response json should contain the returned data from getJoinResp`)
	assert.Equal(t, 0, int(resp[_VERSION].(float64)), `response json should contain the version number`)
	assert.Nil(t, s.Values[_USER_ID], `session should be completely cleared`)
	assert.Nil(t, s.Values[_ENTITY_ID], `session should be completely cleared`)
	assert.Nil(t, s.Values[_ENTITY], `session should be completely cleared`)
}

func Test_poll_with_entity_store_read_error(t *testing.T) {
	w, r := setup(nil, nil, nil, _POLL, `{"`+_ID+`": "test_entity_id", "`+_VERSION+`": 0}`)
	tes.readErr = errors.New(`test_read_error`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_read_error\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_poll_with_request_missing_id(t *testing.T) {
	w, r := setup(nil, nil, nil, _POLL, `{}`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, _ID + " value must be included in request\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_poll_with_request_nonstring_id(t *testing.T) {
	w, r := setup(nil, nil, nil, _POLL, `{"`+_ID+`": true}`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, _ID + " must be a string value\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_poll_with_request_missing_version(t *testing.T) {
	w, r := setup(nil, nil, nil, _POLL, `{"`+_ID+`": "test_entity_id"}`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, _VERSION + " value must be included in request\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_poll_with_request_nonnumber_version(t *testing.T) {
	w, r := setup(nil, nil, nil, _POLL, `{"`+_ID+`": "test_entity_id", "`+_VERSION+`": true}`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, _VERSION + " must be a number value\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_act_success(t *testing.T) {
	w, r := setup(nil, func(userId string, entity Entity)Json{return Json{"test": "yo"}}, func(json Json, userId string, e Entity)error{return nil}, _ACT, ``)
	tes.Create()
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{getVersion:func()int{return 0}}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	resp := Json{}
	readTestJson(w, &resp)
	assert.Equal(t, `yo`, resp[`test`].(string), `response json should contain the returned data from getEntityChangeResp`)
	assert.Equal(t, 0, int(resp[_VERSION].(float64)), `response json should contain the version number`)
	assert.Equal(t, `test_pre_set_user_id`, s.Values[_USER_ID], `session should contain same userId`)
	assert.Equal(t, `test_pre_set_entity_id`, s.Values[_ENTITY_ID], `session should contain same entityId`)
	assert.NotEqual(t, entity, s.Values[_ENTITY], `session entity should not be it's original value`)
	assert.Equal(t, tes.entity, s.Values[_ENTITY], `session entity should be updated to the stores entity`)
}

func Test_act_to_inactive_entity(t *testing.T) {
	w, r := setup(nil, func(userId string, entity Entity)Json{return Json{"test": "yo"}}, func(json Json, userId string, e Entity)error{return nil}, _ACT, ``)
	tes.Create()
	tes.entity.isActive = func()bool{return false}
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	resp := Json{}
	readTestJson(w, &resp)
	assert.Equal(t, `yo`, resp[`test`].(string), `response json should contain the returned data from getEntityChangeResp`)
	assert.Equal(t, 0, int(resp[_VERSION].(float64)), `response json should contain the version number`)
	assert.Nil(t, s.Values[_USER_ID], `session should have been cleared`)
	assert.Nil(t, s.Values[_ENTITY_ID], `session should have been cleared`)
	assert.Nil(t, s.Values[_ENTITY], `session should have been cleared`)
}

func Test_act_with_empty_session(t *testing.T) {
	w, r := setup(nil, nil, nil, _ACT, ``)

	tr.ServeHTTP(w, r)

	assert.Equal(t, "no entity in session\n", w.Body.String(), `response body should have error message`)
	assert.Equal(t, 500, w.Code, `response code should be 500`)
}

func Test_act_with_performAct_error_on_session_entity(t *testing.T) {
	w, r := setup(nil, nil, func(json Json, userId string, e Entity)error{return errors.New(`test_perform_act_error`)}, _ACT, ``)
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_perform_act_error\n", w.Body.String(), `response body should have error message`)
	assert.Equal(t, 500, w.Code, `response code should be 500`)
}

func Test_act_with_read_error(t *testing.T) {
	w, r := setup(nil, nil, func(json Json, userId string, e Entity)error{return nil}, _ACT, ``)
	tes.readErr = errors.New(`test_read_error`)
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_read_error\n", w.Body.String(), `response body should have error message`)
	assert.Equal(t, 500, w.Code, `response code should be 500`)
}

func Test_act_with_performAct_error_on_stored_entity(t *testing.T) {
	callCount := 0
	w, r := setup(nil, nil, func(json Json, userId string, e Entity)error{
		if callCount == 0 {
			callCount++
			return nil
		}
		return errors.New(`test_perform_act_error`)
	}, _ACT, ``)
	tes.Create()
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_perform_act_error\n", w.Body.String(), `response body should have error message`)
	assert.Equal(t, 500, w.Code, `response code should be 500`)
}

func Test_act_with_update_error(t *testing.T) {
	w, r := setup(nil, nil, func(json Json, userId string, e Entity)error{return nil}, _ACT, ``)
	tes.Create()
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity
	tes.updateErr = errors.New(`test_update_error`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_update_error\n", w.Body.String(), `response body should have error message`)
	assert.Equal(t, 500, w.Code, `response code should be 500`)
}

func Test_leave_without_session(t *testing.T) {
	w, r := setup(nil, nil, nil, _LEAVE, ``)

	tr.ServeHTTP(w, r)

	assert.Equal(t, ``, w.Body.String(), `response body should be empty`)
	assert.Equal(t, 200, w.Code, `response code should be 200`)
}

func Test_leave_with_session(t *testing.T) {
	w, r := setup(nil, nil, nil, _LEAVE, ``)
	tes.Create()
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	assert.Equal(t, ``, w.Body.String(), `response body should be empty`)
	assert.Equal(t, 200, w.Code, `response code should be 200`)
}

func Test_leave_with_session_entity_unregister_user_error(t *testing.T) {
	w, r := setup(nil, nil, nil, _LEAVE, ``)
	tes.Create()
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{unregisterUser:func(s string)error{return errors.New(`test_unregister_user_error`)}}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_unregister_user_error\n", w.Body.String(), `response body should contain error`)
	assert.Equal(t, 500, w.Code, `response code should be 500`)
}

func Test_leave_with_read_error(t *testing.T) {
	w, r := setup(nil, nil, nil, _LEAVE, ``)
	tes.Create()
	tes.readErr = errors.New(`test_read_error`)
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_read_error\n", w.Body.String(), `response body should contain error`)
	assert.Equal(t, 500, w.Code, `response code should be 500`)
}

func Test_leave_with_stored_entity_unregister_user_error(t *testing.T) {
	w, r := setup(nil, nil, nil, _LEAVE, ``)
	tes.Create()
	tes.entity.unregisterUser = func(s string)error{return errors.New(`test_unregister_user_error`)}
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_unregister_user_error\n", w.Body.String(), `response body should contain error`)
	assert.Equal(t, 500, w.Code, `response code should be 500`)
}

func Test_leave_with_update_error(t *testing.T) {
	w, r := setup(nil, nil, nil, _LEAVE, ``)
	tes.Create()
	tes.updateErr = errors.New(`test_update_error`)
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_update_error\n", w.Body.String(), `response body should contain error`)
	assert.Equal(t, 500, w.Code, `response code should be 500`)
}

/**
 * helpers
 */

var tr *mux.Router

func readTestJson(w *httptest.ResponseRecorder, obj interface{}) error{
	return js.Unmarshal(w.Body.Bytes(), obj)
}

func setup(gjr GetJoinResp, gecr GetEntityChangeResp, pa PerformAct, path string, reqJson string) (*httptest.ResponseRecorder, *http.Request){
	tss = &testSessionStore{}
	tes = &testEntityStore{}
	tr = mux.NewRouter()
	Route(tr, tss, `test_session`, &testEntity{}, tes, gjr, gecr, pa)
	w := httptest.NewRecorder()
	var r *http.Request
	if reqJson != `` {
		r, _ = http.NewRequest(`POST`, path, bytes.NewBuffer([]byte(reqJson)))
	} else {
		r, _ = http.NewRequest(`POST`, path, nil)
	}
	return w, r
}

/**
 * Session
 */

var tss *testSessionStore

type testSessionStore struct{
	session *sessions.Session
}

func (tss *testSessionStore) Get(r *http.Request, sessName string) (*sessions.Session, error) {
	if tss.session == nil {
		tss.session = sessions.NewSession(tss, sessName)
	}
	return tss.session, nil
}

func (tss *testSessionStore) New(r *http.Request, sessName string) (*sessions.Session, error) {
	if tss.session == nil {
		tss.session = sessions.NewSession(tss, sessName)
	}
	return tss.session, nil
}

func (tss *testSessionStore) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	return nil
}

/**
 * Entity
 */

var tes *testEntityStore

type testEntityStore struct {
	entityId string
	entity *testEntity
	createErr error
	readErr error
	updateErr error
	update func(entityId string, entity Entity) error
}

func (tes *testEntityStore) Create() (entityId string, entity Entity, err error) {
	if tes.entity == nil {
		tes.entity = &testEntity{}
	}
	if tes.entityId == `` {
		tes.entityId = `test_entity_id`
	}
	return tes.entityId, tes.entity, tes.createErr
}

func (tes *testEntityStore) Read(entityId string) (Entity, error) {
	return tes.entity, tes.readErr
}

func (tes *testEntityStore) Update(entityId string, entity Entity) error {
	tes.entity = entity.(*testEntity)
	if tes.update != nil {
		return tes.update(entityId, entity)
	}
	return tes.updateErr
}

type testEntity struct {
	getVersion func() int
	isActive func() bool
	createdBy func() string
	registerNewUser func() (string, error)
	unregisterUser func(string) error
	kick func() bool
}

func (te *testEntity) GetVersion() int {
	if te.getVersion != nil {
		return te.getVersion()
	}
	return 0
}

func (te *testEntity) IsActive() bool {
	if te.isActive != nil {
		return te.isActive()
	}
	return true
}

func (te *testEntity) CreatedBy() string {
	if te.createdBy != nil {
		return te.createdBy()
	}
	return `test_creator_user_id`
}

func (te *testEntity) RegisterNewUser() (string, error) {
	if te.registerNewUser != nil {
		return te.registerNewUser()
	}
	return `test_user_id`, nil
}

func (te *testEntity) UnregisterUser(userId string) error {
	if te.unregisterUser != nil {
		return te.unregisterUser(userId)
	}
	return nil
}

func (te *testEntity) Kick() bool {
	if te.kick != nil {
		return te.kick()
	}
	return false
}
