package gus

import(
	`github.com/0xor1/sus`
	`github.com/qedus/nds`
	`golang.org/x/net/context`
	`google.golang.org/appengine/datastore`
)

// Creates and configures a store that stores entities in Google AppEngines memcache and datastore.
// github.com/qedus/nds is used for strongly consistent automatic caching.
func NewGaeStore(kind string, ctx context.Context, idf sus.IdFactory, vf sus.VersionFactory) sus.Store {
	getKey := func(ctx context.Context, id string) *datastore.Key {
		return datastore.NewKey(ctx, kind, id, 0, nil)
	}

	getMulti := func(ids []string) (vs []sus.Version, err error) {
		count := len(ids)
		vs = make([]sus.Version, count, count)
		ks := make([]*datastore.Key, count, count)
		for i := 0; i < count; i++ {
			vs[i] = vf()
			ks[i] = getKey(ctx, ids[i])
		}
		err = nds.GetMulti(ctx, ks, vs)
		return
	}

	putMulti := func(ids []string, vs []sus.Version) (err error) {
		count := len(ids)
		ks := make([]*datastore.Key, count, count)
		for i := 0; i < count; i++ {
			ks[i] = getKey(ctx, ids[i])
		}
		_, err = nds.PutMulti(ctx, ks, vs)
		return
	}

	delMulti := func(ids []string) error {
		count := len(ids)
		ks := make([]*datastore.Key, count, count)
		for i := 0; i < count; i++ {
			ks[i] = getKey(ctx, ids[i])
		}
		return nds.DeleteMulti(ctx, ks)
	}

	isNonExtantError := func(err error) bool {
		return err.Error() == datastore.ErrNoSuchEntity.Error()
	}

	rit := func(tran sus.Transaction) error {
		return nds.RunInTransaction(ctx, func(ctx context.Context)error{
			return tran()
		}, &datastore.TransactionOptions{XG:true})
	}

	return sus.NewStore(getMulti, putMulti, delMulti, idf, vf, isNonExtantError,rit)
}
