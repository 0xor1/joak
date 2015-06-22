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
}

type EntityFactory func()Entity

func now() time.Time {
	return time.Now().UTC()
}

type gaeStoreObj struct{
	Entity	 				`datastore:",noindex"`
	DeleteAfter time.Time	`datastore:""`
}

func newGaeStore(kind string, ctx context.Context, ef EntityFactory, deleteAfterDur time.Duration, clearOutDur time.Duration) (oak.EntityStore, error) {
	if kind == `` {
		return nil, errors.New(`kind must not be an empty string`)
	}
	if deleteAfterDur.Seconds() <= 0 {
		return nil, errors.New(`deleteAfterDur must be a positive time.Duration`)
	}
	if clearOutDur.Seconds() <= 0 {
		return nil, errors.New(`clearOutDur must be a positive time.Duration`)
	}

	var lastGaeClearOut time.Time
	var mtx sync.Mutex

	pre := func() {
		myLastGaeClearOutInst := lastGaeClearOut
		if lastGaeClearOut.IsZero() || time.Since(lastGaeClearOut) >= clearOutDur {
			mtx.Lock()
			if lastGaeClearOut != myLastGaeClearOutInst {
				mtx.Unlock()
				return
			}
			lastGaeClearOut = now()
			mtx.Unlock()
			q := datastore.NewQuery(kind).Filter(`DeleteAfter <=`, now()).KeysOnly()
			keys := []*datastore.Key{}
			for iter := q.Run(context.Background()); ; {
				key, err := iter.Next(nil)
				if err == datastore.Done {
					break
				}
				if err != nil {
					return
				}
				keys = append(keys, key)
			}
			nds.DeleteMulti(context.Background(), keys)
		}
		return
	}

	return &entityStore{isForGae: true, deleteAfter: deleteAfterDur, preprocess: pre, inner: gus.NewGaeStore(kind, ctx, sid.Uuid, func()sus.Version{return &gaeStoreObj{Entity: ef(), DeleteAfter: now().Add(deleteAfterDur)}})}, nil
}

func newMemoryStore(ef EntityFactory) oak.EntityStore {
	pre := func(){}
	return &entityStore{isForGae: false, preprocess: pre, inner: sus.NewJsonMemoryStore(sid.Uuid, func()sus.Version{return ef()})}
}

type entityStore struct {
	isForGae	bool
	deleteAfter	time.Duration
	preprocess  func()
	inner 		sus.Store
}

func (es *entityStore) Create() (string, oak.Entity, error) {
	go es.preprocess()
	id, v, err := es.inner.Create()
	var e Entity
	if err == nil && v != nil {
		e = v.(Entity)
	}
	return id, e, err
}

func (es *entityStore) Read(entityId string) (oak.Entity, error) {
	go es.preprocess()
	v, err := es.inner.Read(entityId)
	var e Entity
	if err == nil && v != nil {
		e = v.(Entity)
	}
	return e, err
}

func (es *entityStore) Update(entityId string, entity oak.Entity) (error) {
	go es.preprocess()
	if es.isForGae && es.deleteAfter.Seconds() > 0{
		gso, ok := entity.(*gaeStoreObj)
		if ok {
			gso.DeleteAfter = now().Add(es.deleteAfter)
		}
	}
	e, _ := entity.(Entity)
	return es.inner.Update(entityId, e)
}

func RouteLocalTest(router *mux.Router, ef EntityFactory, sessionMaxAge int, sessionName string, entity Entity, getJoinResp oak.GetJoinResp, getEntityChangeResp oak.GetEntityChangeResp, performAct oak.PerformAct){
	ss := sessions.NewCookieStore()
	ss.Options.HttpOnly = false
	ss.Options.MaxAge = sessionMaxAge
	oak.Route(router, ss, sessionName, entity, newMemoryStore(ef), getJoinResp, getEntityChangeResp, performAct)
}

func RouteGaeProd(router *mux.Router, ctx context.Context, ef EntityFactory, sessionMaxAge int, sessionName string, entity Entity, getJoinResp oak.GetJoinResp, getEntityChangeResp oak.GetEntityChangeResp, performAct oak.PerformAct, kind string, deleteAfterDuration time.Duration, clearOutDur time.Duration, newAuthKey string, newCryptKey string, oldAuthKey string, oldCryptKey string) error {
	ss := sessions.NewCookieStore([]byte(newAuthKey), []byte(newCryptKey), []byte(oldAuthKey), []byte(oldCryptKey))
	ss.Options.HttpOnly = true
	ss.Options.MaxAge = sessionMaxAge
	es, err := newGaeStore(kind, ctx, ef, deleteAfterDuration, clearOutDur)
	if err != nil {
		return err
	}
	oak.Route(router, ss, sessionName, entity, es, getJoinResp, getEntityChangeResp, performAct)
	return nil
}