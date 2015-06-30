package joak

import(
	`time`
	`sync`
	`errors`
	`github.com/0xor1/oak`
	`github.com/0xor1/sus`
	`github.com/0xor1/gus`
	`github.com/0xor1/sid`
	`github.com/qedus/nds`
	`github.com/gorilla/mux`
	`golang.org/x/net/context`
	`github.com/gorilla/sessions`
	`google.golang.org/appengine/datastore`
)

type Entity interface{
	oak.Entity
	IncrementVersion()
	DecrementVersion()
	SetDeleteAfter(time.Time)
}

type EntityFactory func()Entity

func now() time.Time {
	return time.Now().UTC()
}

func newGaeStore(kind string, ctx context.Context, ef EntityFactory, deleteAfter time.Duration, clearOutAfter time.Duration) (oak.EntityStore, error) {
	if kind == `` {
		return nil, errors.New(`kind must not be an empty string`)
	}
	if deleteAfter.Seconds() <= 0 {
		return nil, errors.New(`deleteAfter must be a positive time.Duration`)
	}
	if clearOutAfter.Seconds() <= 0 {
		return nil, errors.New(`clearOutAfter must be a positive time.Duration`)
	}

	var lastClearOut time.Time
	var mtx sync.Mutex

	clearOut := func() {
		myLastClearOutInst := lastClearOut
		if lastClearOut.IsZero() || time.Since(lastClearOut) >= clearOutAfter {
			mtx.Lock()
			if lastClearOut != myLastClearOutInst {
				mtx.Unlock()
				return
			}
			lastClearOut = now()
			mtx.Unlock()
			q := datastore.NewQuery(kind).Filter(`DeleteAfter <=`, now()).KeysOnly()
			keys := []*datastore.Key{}
			for iter := q.Run(ctx);; {
				key, err := iter.Next(nil)
				if err == datastore.Done {
					break
				}
				if err != nil {
					return
				}
				keys = append(keys, key)
			}
			nds.DeleteMulti(ctx, keys)
		}
	}

	return &entityStore{deleteAfter, clearOut, gus.NewGaeStore(kind, ctx, sid.Uuid, func()sus.Version{
		e := ef()
		e.SetDeleteAfter(now().Add(deleteAfter))
		return e
	})}, nil
}

func newMemoryStore(ef EntityFactory) oak.EntityStore {
	var deleteAfter time.Duration
	return &entityStore{deleteAfter, func(){}, sus.NewJsonMemoryStore(sid.Uuid, func()sus.Version{return ef()})}
}

type entityStore struct {
	deleteAfter time.Duration
	clearOut  	func()
	inner 		sus.Store
}

func (es *entityStore) Create() (string, oak.Entity, error) {
	go es.clearOut()
	id, v, err := es.inner.Create()
	var e Entity
	if err == nil && v != nil {
		e = v.(Entity)
	}
	return id, e, err
}

func (es *entityStore) Read(entityId string) (oak.Entity, error) {
	go es.clearOut()
	v, err := es.inner.Read(entityId)
	var e Entity
	if err == nil && v != nil {
		e = v.(Entity)
	}
	return e, err
}

func (es *entityStore) Update(entityId string, entity oak.Entity) (error) {
	go es.clearOut()
	e, ok := entity.(Entity)
	if ok {
		e.SetDeleteAfter(now().Add(es.deleteAfter))
	}
	return es.inner.Update(entityId, e)
}

func RouteLocalTest(router *mux.Router, ef EntityFactory, sessionMaxAge int, sessionName string, newAuthKey string, newCryptKey string, oldAuthKey string, oldCryptKey string, entity Entity, getJoinResp oak.GetJoinResp, getEntityChangeResp oak.GetEntityChangeResp, performAct oak.PerformAct){
	sessionStore := initCookieSessionStore(sessionMaxAge, newAuthKey, newCryptKey, oldAuthKey, oldCryptKey)
	oak.Route(router, sessionStore, sessionName, entity, newMemoryStore(ef), getJoinResp, getEntityChangeResp, performAct)
}

func RouteGaeProd(router *mux.Router, ef EntityFactory, sessionMaxAge int, sessionName string, newAuthKey string, newCryptKey string, oldAuthKey string, oldCryptKey string, entity Entity, getJoinResp oak.GetJoinResp, getEntityChangeResp oak.GetEntityChangeResp, performAct oak.PerformAct, deleteAfter time.Duration, clearOutAfter time.Duration, kind string, ctx context.Context) error {
	sessionStore := initCookieSessionStore(sessionMaxAge, newAuthKey, newCryptKey, oldAuthKey, oldCryptKey)
	es, err := newGaeStore(kind, ctx, ef, deleteAfter, clearOutAfter)
	if err != nil {
		return err
	}
	oak.Route(router, sessionStore, sessionName, entity, es, getJoinResp, getEntityChangeResp, performAct)
	return nil
}

func initCookieSessionStore(sessionMaxAge int, newAuthKey string, newCryptKey string, oldAuthKey string, oldCryptKey string) sessions.Store {
	ss := sessions.NewCookieStore([]byte(newAuthKey), []byte(newCryptKey), []byte(oldAuthKey), []byte(oldCryptKey))
	ss.Options.HttpOnly = true
	ss.Options.MaxAge = sessionMaxAge
	return ss
}