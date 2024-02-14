package caller

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type a struct {
}

func (st *a) twoHops() Ptr {
	return func() Ptr { return NewCaller(TwoHopsCallerDepth) }()
}

func (st *a) oneHop() Ptr {
	return NewCaller(FnCallerDepth)
}

func (st *a) noHop() Ptr {
	return NewCaller(SelfCallerDepth)
}

func (st *a) lotHop() Ptr {
	return func() Ptr { return func() Ptr { return func() Ptr { return Self() }() }() }()
}

func TestCallerSelf(t *testing.T) {
	c := Self()
	c1 := func() *Caller { return Self() }()
	c2 := func() *Caller { return Self() }()
	assert.Equal(t, Caller{"caller.TestCallerSelf", 0, nil}.String(), c.String())
	assert.Equal(t, Caller{"caller.TestCallerSelf.func1", 0, nil}.String(), c1.String())
	assert.Equal(t, Caller{"caller.TestCallerSelf.func2", 0, nil}.String(), c2.String())

	one := &a{}
	oneC := one.oneHop().String()
	two := &a{}
	twoC := two.twoHops().String()
	none := &a{}
	noneC := none.noHop().String()
	lot := &a{}
	lotC := lot.lotHop().String()
	assert.Equal(t, Caller{"caller.TestCallerSelf", 0, nil}.String(), oneC)
	assert.Equal(t, Caller{"caller.TestCallerSelf", 0, nil}.String(), twoC)
	assert.Equal(t, Caller{"caller.a.noHop", 0, nil}.String(), noneC)
	assert.Equal(t, Caller{"caller.a.lotHop.a.lotHop.func1.func2.1", 0, nil}.String(), lotC)
}
