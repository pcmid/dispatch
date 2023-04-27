package dispatch

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	DomainTreeMatchALl = 0x01
)

type Domain string

type DomainTree struct {
	mark uint8
	sub  domainMap
}

type domainMap map[Domain]*DomainTree

func (d Domain) Next() Domain {
	if pointIndex := strings.LastIndex(string(d), "."); pointIndex >= 0 {
		return d[:pointIndex]
	}
	return ""
}

func (d Domain) Top() Domain {
	if pointIndex := strings.LastIndex(string(d), "."); pointIndex >= 0 {
		return d[pointIndex+1:]
	} else {
		return d
	}
}

func NewDomainTree() (dt *DomainTree) {
	dt = new(DomainTree)
	dt.sub = make(domainMap)
	return
}

func (dt *DomainTree) Has(d string) bool {
	domain := parseDomain(d)
	if domain == "" {
		return false
	}

	return dt.has(domain)
}

func (dt *DomainTree) has(d Domain) bool {
	if dt.mark&DomainTreeMatchALl == DomainTreeMatchALl {
		return true
	}
	if len(dt.sub) == 0 {
		return false
	}

	if sub, ok := dt.sub[d.Top()]; ok {
		return sub.has(d.Next())
	}
	return false
}

func (dt *DomainTree) Insert(d string) {
	domain := parseDomain(d)
	if domain == "" {
		return
	}
	dt.insert(domain)
}

func (dt *DomainTree) insert(d Domain) {
	if d == "" {
		dt.mark |= DomainTreeMatchALl
		return
	}

	if sub, ok := dt.sub[d.Top()]; ok {
		sub.insert(d.Next())
	} else {
		dt.sub[d.Top()] = NewDomainTree()
		dt.sub[d.Top()].insert(d.Next())
	}
}

func (dt *DomainTree) Merge(other *DomainTree) {
	if other == nil {
		return
	}
	dt.merge(other)
}

func (dt *DomainTree) merge(other *DomainTree) {
	dt.mark |= other.mark

	for sec, sub := range other.sub {
		if _, ok := dt.sub[sec]; !ok {
			dt.sub[sec] = NewDomainTree()
		}

		dt.sub[sec].merge(sub)
	}
}

func NewTreeFromFile(file string) (*DomainTree, error) {
	domainFile, err := os.Open(file)
	if err != nil {
		return nil, Error(err)
	}
	defer domainFile.Close()

	return NewTreeFromReader(domainFile)
}

func NewTreeFromUrl(url string) (*DomainTree, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, Error(err)
	}
	defer resp.Body.Close()

	return NewTreeFromReader(resp.Body)
}

func NewTreeFromReader(reader io.Reader) (*DomainTree, error) {
	dt := NewDomainTree()
	buf := bytes.NewBuffer(nil)
	_, err := buf.ReadFrom(reader)
	if err != nil {
		return nil, Error(err)
	}

	for {
		line, err := buf.ReadBytes('\n')
		if err != nil && err != io.EOF {
			log.Warning(err)
			continue
		}
		dt.Insert(string(line))

		if err == io.EOF {
			break
		}
	}
	return dt, nil
}

func parseDomain(domain string) Domain {
	domain = strings.TrimSpace(domain)
	if strings.HasPrefix(domain, "#") {
		return ""
	}

	// dnsmasq
	if strings.HasPrefix(domain, "server=/") {
		fields := strings.Split(domain, "/")
		if len(fields) != 3 {
			return ""
		}
		domain = fields[1]
	}

	if domain == "" {
		return ""
	}

	domain = strings.TrimPrefix(strings.TrimSuffix(domain, "."), ".")

	return Domain(domain)
}
