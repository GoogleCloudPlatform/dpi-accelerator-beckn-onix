// Package main provides the encrypter plugin for beckn-onix.
package main

import (
	"context"

	"github.com/google/dpi-accelerator-beckn-onix/plugins/encrypter"
	"github.com/beckn/beckn-onix/pkg/plugin/definition"
)

// encrypterProvider implements the definition.encrypterProvider interface.
type encrypterProvider struct{}

func (ep encrypterProvider) New(ctx context.Context, config map[string]string) (definition.Encrypter, func() error, error) {
	return encrypter.New(ctx)
}

// Provider is the exported symbol that the plugin manager will look for.
var Provider = encrypterProvider{}

func main() {}