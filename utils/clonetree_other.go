//go:build !darwin && !linux

package utils

import "errors"

func cloneEntry(src, dst string) error { return errors.New("clone not supported") }
