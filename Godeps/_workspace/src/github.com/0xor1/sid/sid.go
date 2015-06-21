package sid

import(
	`code.google.com/p/go-uuid/uuid`
	`labix.org/v2/mgo/bson`
)

// Returns a new random version 4 UUID string.
func Uuid() string {
	return uuid.New()
}

// Returns a new objectId hex string.
func ObjectId() string {
	return bson.NewObjectId().Hex()
}
