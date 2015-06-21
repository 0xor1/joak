package sid

import(
	`regexp`
	`testing`
	`github.com/stretchr/testify/assert`
)

func Test_Uuid(t *testing.T){
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	assert.True(t, re.MatchString(Uuid()), `Uuid should return a valid uuid string`)
}

func Test_ObjectId(t *testing.T){
	re := regexp.MustCompile(`^[0-9a-f]{24}$`)
	assert.True(t, re.MatchString(ObjectId()), `ObjectId should return a valid objectId hex string`)
}
