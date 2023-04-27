package dispatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomain_TopNext(t *testing.T) {
	domain := parseDomain("a.b")
	assert.EqualValues(t, domain.Top(), "b")
	assert.EqualValues(t, domain.Next(), "a")

	domain = domain.Next()
	assert.EqualValues(t, domain.Top(), "a")
	assert.EqualValues(t, domain.Next(), "")

	domain = domain.Next()
	assert.EqualValues(t, domain.Top(), "")
	assert.EqualValues(t, domain.Next(), "")
}

func TestDomainTree_InsertHas(t *testing.T) {
	dt := NewDomainTree()

	dt.Insert(".")
	assert.False(t, dt.Has(""))
	assert.True(t, dt.Has("a"))

	dt = NewDomainTree()
	dt.Insert("b.c")
	assert.False(t, dt.Has(""))
	assert.False(t, dt.Has("c"))
	assert.True(t, dt.Has("b.c"))
	assert.True(t, dt.Has("a.b.c"))
}

func TestDomainTree_Merge(t *testing.T) {
	dt := NewDomainTree()
	other := NewDomainTree()

	other.Insert("b.c")
	other.Insert("d")

	dt.Merge(other)

	assert.False(t, dt.Has(""))
	assert.False(t, dt.Has("c"))
	assert.True(t, dt.Has("b.c"))
	assert.True(t, dt.Has("a.b.c"))
	assert.True(t, dt.Has("a.d"))
}

func TestNewTreeFromUrl(t *testing.T) {
	dt, err := NewTreeFromUrl("https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/accelerated-domains.china.conf")
	if err != nil {
		t.Fatal(err)
	}
	_ = dt
}

func TestNewTreeFromFile(t *testing.T) {
	dt, err := NewTreeFromFile("test/domains.txt")
	if err != nil {
		t.Fatal(err)
	}
	_ = dt
}
