package oak

import(
	`errors`
	`strings`
	`net/http`
	`encoding/gob`
	js `encoding/json`
	`github.com/gorilla/mux`
	`github.com/gorilla/sessions`
)

const (
	_CREATE = `/create`
	_JOIN 	= `/join`
	_POLL 	= `/poll`
	_ACT 	= `/act`
	_LEAVE 	= `/leave`

	_USER_ID	= `userId`
	_ENTITY_ID	= `entityId`
	_ENTITY		= `entity`

	_ID			= `id`
	_VERSION	= `v`
)

type EntityStore interface{
	Create() (entityId string, entity Entity, err error)
	Read(entityId string) (entity Entity, err error)
	Update(entityId string, entity Entity) (err error)
}

type EntityStoreFactory func(r *http.Request) EntityStore

type Entity interface {
	GetVersion() int
	IsActive() bool
	CreatedBy() (userId string)
	RegisterNewUser() (userId string, err error)
	UnregisterUser(userId string) error
	Kick() (updated bool)
}

type GetJoinResp func(userId string, e Entity) Json
type GetEntityChangeResp func(userId string, e Entity) Json
type PerformAct func(json Json, userId string, e Entity) (err error)

func Route(router *mux.Router, sessionStore sessions.Store, sessionName string, entity Entity, entityStoreFactory EntityStoreFactory, getJoinResp GetJoinResp, getEntityChangeResp GetEntityChangeResp, performAct PerformAct){
	gob.Register(entity)

	getSession := func(w http.ResponseWriter, r *http.Request) (*session, error) {
		s, err := sessionStore.Get(r, sessionName)

		session := &session{
			writer: w,
			request: r,
			internalSession: s,
		}

		var val interface{}
		var exists bool

		if val, exists = s.Values[_USER_ID]; exists {
			session.userId = val.(string)
		}else{
			session.userId = ``
		}

		if val, exists = s.Values[_ENTITY_ID]; exists {
			session.entityId = val.(string)
		}else{
			session.entityId = ``
		}

		if val, exists = s.Values[_ENTITY]; exists && val != nil {
			session.entity = val.(Entity)
		}else{
			session.entity = nil
		}

		return session, err
	}

	fetchEntity := func(entityId string, entityStore EntityStore) (entity Entity, err error) {
		retryCount := 0
		for {
			entity, err = entityStore.Read(entityId)
			if err == nil {
				if entity.Kick() {
					err = entityStore.Update(entityId, entity)
					if err != nil && retryCount == 0 && strings.Contains(err.Error(), `nonsequential update for entity with id "`+entityId+`"`) {
						err = nil
						retryCount++
						continue
					}
				}
			}
			break
		}
		return
	}

	create := func(w http.ResponseWriter, r *http.Request){
		s, _ := getSession(w, r)
		if s.isNotEngaged() {
			entityStore := entityStoreFactory(r)
			entityId, entity, err := entityStore.Create()
			if err != nil {
				writeError(w, err)
				return
			}
			s.set(entity.CreatedBy(), entityId, entity)
		}
		writeJson(w, &Json{_ID: s.getEntityId()})
	}

	join := func(w http.ResponseWriter, r *http.Request) {
		entityId, _, err := getRequestData(r, false)
		if err != nil {
			writeError(w, err)
			return
		}

		entityStore := entityStoreFactory(r)
		entity, err := fetchEntity(entityId, entityStore)
		if err != nil {
			writeError(w, err)
			return
		}

		s, _ := getSession(w, r)
		if s.isNotEngaged() && entity.IsActive() {
			if userId, err := entity.RegisterNewUser(); err == nil {
				if err := entityStore.Update(entityId, entity); err == nil {
					//entity was updated successfully this user is now active in this entity
					s.set(userId, entityId, entity)
				}
			}
		}

		respJson := getJoinResp(s.getUserId(), entity)
		respJson[_VERSION] = entity.GetVersion()
		writeJson(w, &respJson)
	}

	poll := func(w http.ResponseWriter, r *http.Request) {
		entityId, version, err := getRequestData(r, true)
		if err != nil {
			writeError(w, err)
			return
		}

		entityStore := entityStoreFactory(r)
		entity, err := fetchEntity(entityId, entityStore)
		if err != nil {
			writeError(w, err)
			return
		}

		if version == entity.GetVersion() {
			writeJson(w, &Json{})
			return
		}

		s, _ := getSession(w, r)
		userId := s.getUserId()
		if s.getEntityId() == entityId {
			if entity.IsActive() {
				s.set(userId, entityId, entity)
			} else {
				s.clear()
			}
		}
		respJson := getEntityChangeResp(userId, entity)
		respJson[_VERSION] = entity.GetVersion()
		writeJson(w, &respJson)
	}

	act := func(w http.ResponseWriter, r *http.Request) {
		s, _ := getSession(w, r)
		userId := s.getUserId()
		sessionEntity := s.getEntity()
		if sessionEntity == nil {
			writeError(w, errors.New(`no entity in session`))
			return
		}

		json := readJson(r)
		err := performAct(json, userId, sessionEntity)
		if err != nil {
			writeError(w, err)
			return
		}

		entityStore := entityStoreFactory(r)
		entityId := s.getEntityId()
		entity, err := fetchEntity(entityId, entityStore)
		if err != nil {
			writeError(w, err)
			return
		}

		if err = performAct(json, userId, entity); err != nil {
			writeError(w, err)
			return
		}

		if err = entityStore.Update(entityId, entity); err != nil {
			writeError(w, err)
			return
		}

		if entity.IsActive() {
			s.set(s.getUserId(), entityId, entity)
		} else {
			s.clear()
		}
		respJson := getEntityChangeResp(userId, entity)
		respJson[_VERSION] = entity.GetVersion()
		writeJson(w, &respJson)
	}

	leave := func(w http.ResponseWriter, r *http.Request) {
		s, _ := getSession(w, r)
		entityId := s.getEntityId()
		sessionEntity := s.getEntity()
		if sessionEntity == nil{
			s.clear()
			return
		}

		err := sessionEntity.UnregisterUser(s.getUserId())
		if err != nil {
			writeError(w, err)
			return
		}

		entityStore := entityStoreFactory(r)
		entity, err := entityStore.Read(entityId)
		if err != nil {
			writeError(w, err)
			return
		}

		err = entity.UnregisterUser(s.getUserId())
		if err != nil {
			writeError(w, err)
			return
		}

		err = entityStore.Update(entityId, entity)
		if err != nil {
			writeError(w, err)
			return
		}

		s.clear()
	}

	router.Path(_CREATE).HandlerFunc(create)
	router.Path(_JOIN).HandlerFunc(join)
	router.Path(_POLL).HandlerFunc(poll)
	router.Path(_ACT).HandlerFunc(act)
	router.Path(_LEAVE).HandlerFunc(leave)
}

type session struct{
	writer http.ResponseWriter
	request *http.Request
	internalSession *sessions.Session
	userId string
	entityId string
	entity Entity
}

func (s *session) set(userId string, entityId string, entity Entity) error {
	s.userId = userId
	s.entityId = entityId
	s.entity = entity
	s.internalSession.Values = map[interface{}]interface{}{
		_USER_ID: userId,
		_ENTITY_ID: entityId,
		_ENTITY: entity,
	}
	return sessions.Save(s.request, s.writer)
}

func (s *session) clear() error {
	s.userId = ``
	s.entityId = ``
	s.entity = nil
	s.internalSession.Values = map[interface{}]interface{}{}
	return sessions.Save(s.request, s.writer)
}

func (s *session) isNotEngaged() bool {
	return s.entity == nil || !s.entity.IsActive()
}

func (s *session) getUserId() string {
	return s.userId
}

func (s *session) getEntityId() string {
	return s.entityId
}

func (s *session) getEntity() Entity {
	return s.entity
}

type Json map[string]interface{}

func writeJson(w http.ResponseWriter, obj interface{}) error{
	js, err := js.Marshal(obj)
	w.Header().Set(`Content-Type`, `application/json`)
	w.Write(js)
	return err
}

func readJson(r *http.Request) Json {
	json := Json{}
	if r.Body == nil {
		return json
	}
	decoder := js.NewDecoder(r.Body)
	decoder.Decode(&json)
	return json
}

func writeError(w http.ResponseWriter, err error){
	http.Error(w, err.Error(), 500)
}

func getRequestData(r *http.Request, isForPoll bool) (entityId string, version int, err error) {
	reqJson := readJson(r)
	if idParam, exists := reqJson[_ID]; exists {
		if id, ok := idParam.(string); ok {
			entityId = id
			if isForPoll {
				if versionParam, exists := reqJson[_VERSION]; exists {
					if v, ok := versionParam.(float64); ok {
						version = int(v)
					} else {
						err = errors.New(_VERSION + ` must be a number value`)
					}
				} else {
					err = errors.New(_VERSION + ` value must be included in request`)
				}
			}
		} else {
			err = errors.New(_ID + ` must be a string value`)
		}
	} else {
		err = errors.New(_ID +` value must be included in request`)
	}
	return
}
