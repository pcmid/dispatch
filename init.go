package dispatch

import (
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var (
	version = ""
	commit  = ""
)

const (
	_PluginName_ = "dispatch"
)

var (
	log = clog.NewWithPlugin(_PluginName_)
)
