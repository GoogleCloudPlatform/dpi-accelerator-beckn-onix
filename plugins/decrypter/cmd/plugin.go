// Package main provides a decrypter plugin for the Beckn-Onix platform.
package main

import (
	"context"

	"github.com/google/dpi-accelerator-beckn-onix/plugins/decrypter"
	"github.com/beckn/beckn-onix/pkg/plugin/definition"
)

// decrypterProvider implements the definition.decrypterProvider interface.
type decrypterProvider struct{}

// New creates a new Decrypter instance using the provided configuration.
func (dp decrypterProvider) New(ctx context.Context, config map[string]string) (definition.Decrypter, func() error, error) {
	return decrypter.New(ctx)
}

// Provider is the exported symbol that the plugin manager will look for.
var Provider = decrypterProvider{}

func main() {}
