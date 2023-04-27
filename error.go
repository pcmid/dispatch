package dispatch

import "github.com/coredns/coredns/plugin"

func Error(err error) error {
	return plugin.Error(_PluginName_, err)
}
