//go:build android

package eletrocromo

import (
	"context"
	"errors"
)

func (a *App) runDesktop(context.Context, context.CancelFunc) error {
	return errors.New("desktop web view is not available on android")
}

func launchBrowserURL(context.Context, string, string) error {
	return errors.New("desktop web view is not available on android")
}
