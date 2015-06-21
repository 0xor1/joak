package sus

import(
	`testing`
	`github.com/stretchr/testify/assert`
)

func Test_versionImpl(t *testing.T){
	v := NewVersion()
	assert.Equal(t, 0, v.getVersion(), `version should init to 0`)
	assert.Equal(t, 0, v.GetVersion(), `exported GetVersion() should return same as private getVersion()`)
	v.incrementVersion()
	assert.Equal(t, 1, v.getVersion(), `increment should increment by 1`)
	assert.Equal(t, 1, v.GetVersion(), `exported GetVersion() should return same as private getVersion()`)
	v.incrementVersion()
	assert.Equal(t, 2, v.getVersion(), `increment should increment by 1`)
	assert.Equal(t, 2, v.GetVersion(), `exported GetVersion() should return same as private getVersion()`)
	v.decrementVersion()
	assert.Equal(t, 1, v.getVersion(), `decrement should decrement by 1`)
	assert.Equal(t, 1, v.GetVersion(), `exported GetVersion() should return same as private getVersion()`)
}
